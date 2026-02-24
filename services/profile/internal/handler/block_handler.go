package handler

import (
	"encoding/json"
	"net/http"

	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/middleware"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/service"
)

// BlockHandler exposes HTTP endpoints for block/unblock operations.
type BlockHandler struct{ S *service.BlockService }

// NewBlockHandler creates a new BlockHandler.
func NewBlockHandler(s *service.BlockService) *BlockHandler { return &BlockHandler{s} }

// Block blocks another user and removes them from contacts.
func (h *BlockHandler) Block(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.UserKey).(string)

	var req struct {
		User string `json:"user"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.User == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.S.Block(r.Context(), uid, req.User); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Unblock removes a block on another user.
func (h *BlockHandler) Unblock(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.UserKey).(string)

	var req struct {
		User string `json:"user"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.User == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.S.Unblock(r.Context(), uid, req.User); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
