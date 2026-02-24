package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/middleware"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/service"
)

// ContactHandler exposes HTTP endpoints for contact operations.
type ContactHandler struct{ S *service.ContactService }

// NewContactHandler creates a new ContactHandler.
func NewContactHandler(s *service.ContactService) *ContactHandler { return &ContactHandler{s} }

// Add creates a contact relationship for the authenticated user.
func (h *ContactHandler) Add(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.UserKey).(string)

	var req struct {
		Contact string `json:"contact"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Contact == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.S.Add(r.Context(), uid, req.Contact); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Remove deletes a contact relationship for the authenticated user.
func (h *ContactHandler) Remove(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.UserKey).(string)

	var req struct {
		Contact string `json:"contact"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Contact == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.S.Remove(r.Context(), uid, req.Contact); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List returns a paginated list of contacts for the authenticated user.
func (h *ContactHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.UserKey).(string)

	limit := 20
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	list, err := h.S.List(r.Context(), uid, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list contacts"})
		return
	}

	writeJSON(w, http.StatusOK, list)
}
