package grpc

import (
	"errors"
	"log/slog"

	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func MapError(err error) error {
	if err == nil {
		return nil
	}

	if _, ok := status.FromError(err); ok {
		return err
	}

	switch {
	case errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrInvalidToken):
		return status.Error(codes.Unauthenticated, err.Error())

	case errors.Is(err, domain.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, domain.ErrEmailConflict):
		return status.Error(codes.AlreadyExists, err.Error())

	default:
		slog.Error("internal_grpc_error", "error", err)
		return status.Error(codes.Internal, "internal server error")
	}
}
