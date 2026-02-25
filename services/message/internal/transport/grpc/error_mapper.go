package grpc

import (
	"errors"
	"log"

	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MapError converts a domain error into a gRPC status error.
func MapError(err error) error {
	if err == nil {
		return nil
	}

	// Check if it's already a gRPC status error
	if _, ok := status.FromError(err); ok {
		return err
	}

	switch {
	case errors.Is(err, domain.ErrMessageNotFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, domain.ErrNotParticipant):
		return status.Error(codes.PermissionDenied, err.Error())

	case errors.Is(err, domain.ErrInvalidMessage),
		errors.Is(err, domain.ErrInvalidSequence),
		errors.Is(err, domain.ErrMessageTooLarge),
		errors.Is(err, domain.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())

	default:
		// Log actual error to help debugging
		log.Printf("internal gRPC error: %v", err)
		return status.Error(codes.Internal, "internal server error")
	}
}
