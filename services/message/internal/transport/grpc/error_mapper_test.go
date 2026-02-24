package grpc

import (
	"errors"
	"testing"

	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{
			name:     "Nil error",
			err:      nil,
			wantCode: codes.OK,
		},
		{
			name:     "Conversation not found",
			err:      domain.ErrConversationNotFound,
			wantCode: codes.NotFound,
		},
		{
			name:     "Message not found",
			err:      domain.ErrMessageNotFound,
			wantCode: codes.NotFound,
		},
		{
			name:     "Not participant",
			err:      domain.ErrNotParticipant,
			wantCode: codes.PermissionDenied,
		},
		{
			name:     "Not admin",
			err:      domain.ErrNotAdmin,
			wantCode: codes.PermissionDenied,
		},
		{
			name:     "Invalid message",
			err:      domain.ErrInvalidMessage,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "Direct modification",
			err:      domain.ErrDirectModification,
			wantCode: codes.FailedPrecondition,
		},
		{
			name:     "Last admin removal",
			err:      domain.ErrLastAdmin,
			wantCode: codes.FailedPrecondition,
		},
		{
			name:     "Already gRPC error",
			err:      status.Error(codes.AlreadyExists, "already exists"),
			wantCode: codes.AlreadyExists,
		},
		{
			name:     "Unknown error wrapped",
			err:      errors.New("something went wrong"),
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := MapError(tt.err)
			if tt.err == nil {
				if gotErr != nil {
					t.Errorf("MapError() = %v, want nil", gotErr)
				}
				return
			}

			st, ok := status.FromError(gotErr)
			if !ok {
				t.Errorf("MapError() did not return a gRPC status error")
				return
			}

			if st.Code() != tt.wantCode {
				t.Errorf("MapError() code = %v, want %v", st.Code(), tt.wantCode)
			}
		})
	}
}
