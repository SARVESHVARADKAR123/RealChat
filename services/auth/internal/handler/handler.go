package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/service"
)

// AuthHandler exposes HTTP endpoints for authentication operations.
type AuthHandler struct {
	svc *service.AuthService
	db  *sql.DB
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(s *service.AuthService, db *sql.DB) *AuthHandler {
	return &AuthHandler{svc: s, db: db}
}

// Health returns 200 OK.
func (h *AuthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// Ready checks the DB connection.
func (h *AuthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ready"))
}

// Register creates a new user account.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password required"})
		return
	}

	userID, err := h.svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"user_id": userID})
}

// Login authenticates a user and returns access + refresh tokens.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password required"})
		return
	}

	access, refresh, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"access_token":  access,
		"refresh_token": refresh,
	})
}

// Refresh rotates a refresh token and returns new access + refresh tokens.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Refresh string `json:"refresh"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Refresh == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "refresh token required"})
		return
	}

	access, refresh, err := h.svc.Refresh(r.Context(), req.Refresh)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid refresh token"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"access_token":  access,
		"refresh_token": refresh,
	})
}

// Logout revokes a refresh token.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Refresh string `json:"refresh"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Refresh == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "refresh token required"})
		return
	}

	_ = h.svc.Logout(r.Context(), req.Refresh)
	w.WriteHeader(http.StatusNoContent)
}

// writeJSON writes a JSON response with Content-Type header.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
