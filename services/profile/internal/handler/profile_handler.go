package handler

import (
	"encoding/json"
	"net/http"

	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/middleware"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/model"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/service"
)

// ProfileHandler exposes HTTP endpoints for profile operations.
type ProfileHandler struct{ S *service.ProfileService }

// NewProfileHandler creates a new ProfileHandler.
func NewProfileHandler(s *service.ProfileService) *ProfileHandler { return &ProfileHandler{s} }

// Get returns the authenticated user's profile.
func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.UserKey).(string)

	p, err := h.S.Get(r.Context(), uid)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "profile not found"})
		return
	}

	writeJSON(w, http.StatusOK, p)
}

// Update modifies the authenticated user's profile.
func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.UserKey).(string)

	var req struct {
		DisplayName string `json:"display_name"`
		Bio         string `json:"bio"`
		AvatarURL   string `json:"avatar_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	err := h.S.Update(r.Context(), &model.Profile{
		UserID:      uid,
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		AvatarURL:   req.AvatarURL,
	})

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
