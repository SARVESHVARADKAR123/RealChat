package handlers

import (
	"net/http"
	"strings"
	"time"

	presencev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/presence/v1"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/transport"
)

type PresenceHandler struct {
	client presencev1.PresenceApiClient
}

func NewPresenceHandler(c presencev1.PresenceApiClient) *PresenceHandler {
	return &PresenceHandler{client: c}
}

func (h *PresenceHandler) GetPresence(w http.ResponseWriter, r *http.Request) {
	userIDsStr := r.URL.Query().Get("user_ids")
	if userIDsStr == "" {
		transport.WriteError(w, http.StatusBadRequest, "missing_user_ids", "user_ids query parameter is required")
		return
	}

	userIDs := strings.Split(userIDsStr, ",")

	ctx, cancel := transport.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	resp, err := h.client.GetPresence(ctx, &presencev1.GetPresenceRequest{
		UserIds: userIDs,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusOK, resp)
}
