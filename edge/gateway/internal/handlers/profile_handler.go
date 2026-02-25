package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	profilev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/profile/v1"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/middleware"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/transport"
)

type ProfileHandler struct {
	client profilev1.ProfileApiClient
}

func NewProfileHandler(c profilev1.ProfileApiClient) *ProfileHandler {
	return &ProfileHandler{client: c}
}

func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	reqID := middleware.RequestIDFromContext(r.Context())

	ctx, cancel := transport.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	ctx = transport.WithMeta(ctx, userID, reqID)

	resp, err := h.client.GetProfile(ctx, &profilev1.GetProfileRequest{
		UserId: userID,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusOK, resp)
}

func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	reqID := middleware.RequestIDFromContext(r.Context())

	var req struct {
		DisplayName *string `json:"display_name"`
		AvatarURL   *string `json:"avatar_url"`
		Bio         *string `json:"bio"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	ctx = transport.WithMeta(ctx, userID, reqID)

	slog.Debug("profile update requested", "user_id", userID, "req_id", reqID)

	resp, err := h.client.UpdateProfile(ctx, &profilev1.UpdateProfileRequest{
		UserId:      userID,
		DisplayName: req.DisplayName,
		AvatarUrl:   req.AvatarURL,
		Bio:         req.Bio,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusOK, resp)
}
