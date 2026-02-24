package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	messagingv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/messaging/v1"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/middleware"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/transport"
)

type ReceiptHandler struct {
	client messagingv1.MessagingApiClient
}

func NewReceiptHandler(c messagingv1.MessagingApiClient) *ReceiptHandler {
	return &ReceiptHandler{client: c}
}

func (h *ReceiptHandler) ReadReceipt(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())

	var req struct {
		ConversationId string `json:"conversation_id"`
		Sequence       int64  `json:"sequence"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid json")
		return
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Inject UserID into metadata
	ctx = transport.WithUserID(ctx, userID)

	resp, err := h.client.UpdateReadReceipt(ctx, &messagingv1.UpdateReadReceiptRequest{
		UserId:         userID,
		ConversationId: req.ConversationId,
		ReadSequence:   req.Sequence,
	})

	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusOK, resp)
}
