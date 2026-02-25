package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	messagev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/message/v1"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/middleware"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/transport"
	"github.com/google/uuid"
)

// MessageHandler handles all routes that talk to the message service.
type MessageHandler struct {
	client messagev1.MessageApiClient
}

func NewMessageHandler(m messagev1.MessageApiClient) *MessageHandler {
	return &MessageHandler{client: m}
}

// SendMessage POST /api/messages
func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	reqID := middleware.RequestIDFromContext(r.Context())

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
	if req.Content == "" {
		transport.WriteError(w, http.StatusBadRequest, "missing_content", "content is required")
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
	ctx = transport.WithMeta(ctx, userID, reqID)

	resp, err := h.client.SendMessage(ctx, &messagev1.SendMessageRequest{
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

// SyncMessages GET /api/messages
func (h *MessageHandler) SyncMessages(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	reqID := middleware.RequestIDFromContext(r.Context())

	convID := r.URL.Query().Get("conversation_id")
	if convID == "" {
		transport.WriteError(w, http.StatusBadRequest, "missing_conv_id", "conversation_id query parameter is required")
		return
	}

	var after int64
	var limit int32 = 50
	if val := r.URL.Query().Get("after"); val != "" {
		if _, err := fmt.Sscanf(val, "%d", &after); err != nil {
			transport.WriteError(w, http.StatusBadRequest, "invalid_after", "after must be an integer")
			return
		}
	}
	if val := r.URL.Query().Get("limit"); val != "" {
		var parseLimit int32
		if _, err := fmt.Sscanf(val, "%d", &parseLimit); err == nil && parseLimit > 0 {
			limit = parseLimit
		}
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	ctx = transport.WithMeta(ctx, userID, reqID)

	resp, err := h.client.SyncMessages(ctx, &messagev1.SyncMessagesRequest{
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

// DeleteMessage DELETE /api/messages
func (h *MessageHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	reqID := middleware.RequestIDFromContext(r.Context())

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
	ctx = transport.WithMeta(ctx, userID, reqID)

	_, err := h.client.DeleteMessage(ctx, &messagev1.DeleteMessageRequest{
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
