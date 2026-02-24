package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	messagingv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/messaging/v1"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/middleware"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/transport"
	"github.com/google/uuid"
)

type MessagingHandler struct {
	client messagingv1.MessagingApiClient
}

func NewMessagingHandler(c messagingv1.MessagingApiClient) *MessagingHandler {
	return &MessagingHandler{client: c}
}

func (h *MessagingHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())

	var req struct {
		ConversationID string `json:"conversation_id"`
		Content        string `json:"content"`
		IdempotencyKey string `json:"idempotency_key"`
		Type           string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid json")
		return
	}

	if req.ConversationID == "" {
		transport.WriteError(w, http.StatusBadRequest, "missing_conv_id", "conversation_id is required")
		return
	}

	if req.IdempotencyKey == "" {
		req.IdempotencyKey = uuid.NewString()
	}

	msgType := req.Type
	if msgType == "" {
		msgType = "text"
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Inject UserID into metadata
	ctx = transport.WithUserID(ctx, userID)

	resp, err := h.client.SendMessage(ctx, &messagingv1.SendMessageRequest{
		SenderUserId:   userID,
		ConversationId: req.ConversationID,
		Content:        req.Content,
		IdempotencyKey: req.IdempotencyKey,
		MessageType:    msgType,
	})

	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusOK, resp)
}

func (h *MessagingHandler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())

	var req struct {
		ID           string   `json:"conversation_id"`
		Type         string   `json:"type"`
		DisplayName  string   `json:"display_name"`
		AvatarURL    string   `json:"avatar_url"`
		Participants []string `json:"participant_user_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid json")
		return
	}

	// Ensure actor is the first participant
	participants := req.Participants
	if len(participants) == 0 {
		participants = []string{userID}
	} else if participants[0] != userID {
		participants = append([]string{userID}, participants...)
	}

	convID := req.ID
	if convID == "" {
		convID = uuid.NewString()
	}

	var pbType messagingv1.ConversationType
	if req.Type == "group" {
		pbType = messagingv1.ConversationType_GROUP
	} else {
		pbType = messagingv1.ConversationType_DIRECT
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Inject UserID into metadata
	ctx = transport.WithUserID(ctx, userID)

	resp, err := h.client.CreateConversation(ctx, &messagingv1.CreateConversationRequest{
		ConversationId:     convID,
		Type:               pbType,
		DisplayName:        req.DisplayName,
		AvatarUrl:          req.AvatarURL,
		ParticipantUserIds: participants,
	})

	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusCreated, resp)
}

func (h *MessagingHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	convID := r.URL.Query().Get("id")
	if convID == "" {
		transport.WriteError(w, http.StatusBadRequest, "missing_conv_id", "id query parameter is required")
		return
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	resp, err := h.client.GetConversation(ctx, &messagingv1.GetConversationRequest{
		ConversationId: convID,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusOK, resp)
}

func (h *MessagingHandler) AddParticipant(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())

	var req struct {
		ConversationID string `json:"conversation_id"`
		TargetUserID   string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid json")
		return
	}

	if req.ConversationID == "" {
		transport.WriteError(w, http.StatusBadRequest, "missing_conv_id", "conversation_id is required")
		return
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Inject UserID into metadata
	ctx = transport.WithUserID(ctx, userID)

	_, err := h.client.AddParticipant(ctx, &messagingv1.AddParticipantRequest{
		ConversationId: req.ConversationID,
		ActorUserId:    userID,
		TargetUserId:   req.TargetUserID,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *MessagingHandler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())

	var req struct {
		ConversationID string `json:"conversation_id"`
		TargetUserID   string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid json")
		return
	}

	if req.ConversationID == "" || req.TargetUserID == "" {
		transport.WriteError(w, http.StatusBadRequest, "missing_params", "conversation_id and user_id are required")
		return
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Inject UserID into metadata
	ctx = transport.WithUserID(ctx, userID)

	_, err := h.client.RemoveParticipant(ctx, &messagingv1.RemoveParticipantRequest{
		ConversationId: req.ConversationID,
		ActorUserId:    userID,
		TargetUserId:   req.TargetUserID,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *MessagingHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())

	var req struct {
		ConversationID string `json:"conversation_id"`
		MessageID      string `json:"message_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid json")
		return
	}

	if req.ConversationID == "" || req.MessageID == "" {
		transport.WriteError(w, http.StatusBadRequest, "missing_params", "conversation_id and message_id are required")
		return
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Inject UserID into metadata
	ctx = transport.WithUserID(ctx, userID)

	_, err := h.client.DeleteMessage(ctx, &messagingv1.DeleteMessageRequest{
		ConversationId: req.ConversationID,
		MessageId:      req.MessageID,
		ActorUserId:    userID,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *MessagingHandler) SyncMessages(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	convID := r.URL.Query().Get("conversation_id")
	if convID == "" {
		transport.WriteError(w, http.StatusBadRequest, "missing_conv_id", "conversation_id query parameter is required")
		return
	}

	// Get pagination params from query
	var after int64
	var limit int32 = 50

	if val := r.URL.Query().Get("after"); val != "" {
		fmt.Sscanf(val, "%d", &after)
	}
	if val := r.URL.Query().Get("limit"); val != "" {
		fmt.Sscanf(val, "%d", &limit)
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Inject UserID into metadata
	ctx = transport.WithUserID(ctx, userID)

	resp, err := h.client.SyncMessages(ctx, &messagingv1.SyncMessagesRequest{
		ConversationId: convID,
		AfterSequence:  after,
		PageSize:       limit,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusOK, resp)
}

func (h *MessagingHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())

	ctx, cancel := transport.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Inject UserID into metadata
	ctx = transport.WithUserID(ctx, userID)

	resp, err := h.client.ListConversations(ctx, &messagingv1.ListConversationsRequest{
		UserId: userID,
	})

	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusOK, resp)
}
