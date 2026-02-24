package grpc

import (
	"context"

	authv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/auth/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/service"
)

type Handler struct {
	authv1.UnimplementedAuthApiServer
	svc *service.AuthService
}

func NewHandler(svc *service.AuthService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	userID, err := h.svc.Register(ctx, req.Email, req.Password)
	if err != nil {
		return nil, MapError(err)
	}
	return &authv1.RegisterResponse{
		UserId: userID,
	}, nil
}

func (h *Handler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	access, refresh, err := h.svc.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, MapError(err)
	}

	return &authv1.LoginResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

func (h *Handler) Refresh(ctx context.Context, req *authv1.RefreshRequest) (*authv1.RefreshResponse, error) {
	access, refresh, err := h.svc.Refresh(ctx, req.RefreshToken)
	if err != nil {
		return nil, MapError(err)
	}

	return &authv1.RefreshResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

func (h *Handler) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	err := h.svc.Logout(ctx, req.RefreshToken)
	if err != nil {
		return nil, MapError(err)
	}

	return &authv1.LogoutResponse{}, nil
}
