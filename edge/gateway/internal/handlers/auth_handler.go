package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	authv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/auth/v1"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/middleware"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/transport"
)

type AuthHandler struct {
	client authv1.AuthApiClient
}

func NewAuthHandler(c authv1.AuthApiClient) *AuthHandler {
	return &AuthHandler{client: c}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		transport.WriteError(w, http.StatusBadRequest, "missing_fields", "email and password are required")
		return
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Propagate request ID
	reqID := middleware.RequestIDFromContext(r.Context())
	ctx = transport.WithRequestID(ctx, reqID)

	resp, err := h.client.Login(ctx, &authv1.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		transport.WriteError(w, http.StatusBadRequest, "missing_fields", "email and password are required")
		return
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Propagate request ID
	reqID := middleware.RequestIDFromContext(r.Context())
	ctx = transport.WithRequestID(ctx, reqID)

	resp, err := h.client.Register(ctx, &authv1.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusCreated, resp)
}
