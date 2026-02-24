package grpc

import (
	"errors"
	"log/slog"

	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/model"
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
	case errors.Is(err, model.ErrProfileNotFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, model.ErrInvalidUpdate):
		return status.Error(codes.InvalidArgument, err.Error())

	default:
		slog.Error("internal_grpc_error", "error", err)
		return status.Error(codes.Internal, "internal server error")
	}
}
