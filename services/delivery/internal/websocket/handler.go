package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	conversationv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/conversation/v1"
	messagev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/message/v1"
	presencev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/presence/v1"
	sharedv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/shared/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/observability"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type Handler struct {
	registry       *Registry
	presenceClient presencev1.PresenceApiClient
	convClient     conversationv1.ConversationApiClient
	msgClient      messagev1.MessageApiClient
	instanceID     string
}

type ResumeRequest struct {
	LastSequences map[string]int64 `json:"last_sequences"`
}

func NewHandler(registry *Registry, pc presencev1.PresenceApiClient, cc conversationv1.ConversationApiClient, mc messagev1.MessageApiClient, instanceID string) *Handler {
	return &Handler{
		registry:       registry,
		presenceClient: pc,
		convClient:     cc,
		msgClient:      mc,
		instanceID:     instanceID,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	deviceID := r.URL.Query().Get("device_id")

	if userID == "" || deviceID == "" {
		http.Error(w, "missing user_id or device_id", http.StatusBadRequest)
		return
	}

	log := observability.GetLogger(r.Context())
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("upgrade error", zap.Error(err))
		return
	}

	sessionID := uuid.NewString()
	session := NewSession(sessionID, userID, deviceID, conn)

	// Register session but it's not ready yet
	h.registry.Add(session)

	// Update presence via gRPC
	ctx := context.Background()
	if _, err := h.presenceClient.RegisterSession(ctx, &presencev1.RegisterSessionRequest{
		UserId:     userID,
		DeviceId:   deviceID,
		InstanceId: h.instanceID,
	}); err != nil {
		log.Error("error setting presence online", zap.Error(err))
	}

	StartHeartbeat(h.presenceClient, userID, deviceID, session.Done())

	session.Start()
	log.Info("connected", zap.String("user_id", userID), zap.String("device_id", deviceID))
	observability.WebSocketConnectionsTotal.WithLabelValues("delivery").Inc()

	// Set read deadline and pong handler
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	go h.readLoop(session)
}

func (h *Handler) readLoop(s *Session) {
	defer func() {
		h.registry.Remove(s)
		s.Close()
		log := observability.GetLogger(context.Background())
		if _, err := h.presenceClient.UnregisterSession(context.Background(), &presencev1.UnregisterSessionRequest{
			UserId:   s.UserID,
			DeviceId: s.DeviceID,
		}); err != nil {
			log.Error("presence: fail to unregister", zap.String("user_id", s.UserID), zap.String("device_id", s.DeviceID), zap.Error(err))
		}
		log.Info("disconnected", zap.String("user_id", s.UserID), zap.String("device_id", s.DeviceID))
		observability.WebSocketConnectionsTotal.WithLabelValues("delivery").Dec()
	}()

	// The first message MUST be the resume request (JSON)
	// handleResume handles reading this message and syncing history
	h.handleResume(s)

	for {
		if _, _, err := s.Conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				observability.Log.Error("read loop error", zap.String("user_id", s.UserID), zap.String("device_id", s.DeviceID), zap.Error(err))
			}
			return
		}
	}
}

func (h *Handler) handleResume(s *Session) {
	_, msg, err := s.Conn.ReadMessage()
	if err != nil {
		s.Close()
		return
	}

	// Parse resume JSON
	var req ResumeRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		s.Close()
		return
	}

	// 1. Fetch all conversations for the user to discover new ones
	ctx := metadata.AppendToOutgoingContext(context.Background(), "x-user-id", s.UserID)
	listResp, err := h.convClient.ListConversations(ctx, &conversationv1.ListConversationsRequest{
		UserId: s.UserID,
	})
	if err != nil {
		observability.Log.Error("resume: error listing conversations", zap.String("user_id", s.UserID), zap.Error(err))
	}

	// 2. Build map of all conversations to sync
	toSync := make(map[string]int64)
	// Add from client request
	for cid, seq := range req.LastSequences {
		toSync[cid] = seq
	}
	// Add discovered ones (if not already present)
	if listResp != nil {
		for _, conv := range listResp.GetConversations() {
			if _, ok := toSync[conv.ConversationId]; !ok {
				toSync[conv.ConversationId] = 0 // New conversation caught while offline
			}
		}
	}

	// 3. Sync each conversation (handling pagination)
	for convID, lastSeq := range toSync {
		h.syncConversation(ctx, s, convID, lastSeq)
	}

	// Resume complete
	s.FlushBufferSorted()
}

func (h *Handler) syncConversation(ctx context.Context, s *Session, convID string, lastSeq int64) {
	currentSeq := lastSeq
	for {
		resp, err := h.msgClient.SyncMessages(
			ctx,
			&messagev1.SyncMessagesRequest{
				ConversationId: convID,
				AfterSequence:  currentSeq,
				PageSize:       100,
			},
		)
		if err != nil {
			observability.Log.Error("resume: error syncing messages", zap.String("conversation_id", convID), zap.Error(err))
			break
		}

		msgs := resp.GetMessages()
		if len(msgs) == 0 {
			break
		}

		for _, m := range msgs {
			h.sendMsgAsEvent(s, m)
			if m.Sequence > currentSeq {
				currentSeq = m.Sequence
			}
		}

		if len(msgs) < 100 {
			break // No more pages
		}
	}
}

func (h *Handler) sendMsgAsEvent(s *Session, m *messagev1.Message) {
	// Wrap in MessageSentEvent
	event := &messagev1.MessageSentEvent{
		Message: m,
	}
	eventPayload, err := proto.Marshal(event)
	if err != nil {
		observability.Log.Error("failed to marshal message event", zap.Error(err))
		return
	}

	// Wrap in EventEnvelope
	env := &sharedv1.EventEnvelope{
		EventType:     sharedv1.EventType_EVENT_TYPE_MESSAGE_SENT,
		SchemaVersion: 1,
		OccurredAt:    m.SentAt,
		Payload:       eventPayload,
	}

	payload, err := proto.Marshal(env)
	if err != nil {
		observability.Log.Error("failed to marshal event envelope", zap.Error(err))
		return
	}

	s.TrySend(payload)
}
