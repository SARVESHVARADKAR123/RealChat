package transport

import (
	"context"

	"google.golang.org/grpc/metadata"
)

const HeaderUserID = "x-user-id"

// WithUserID injects the user ID into the gRPC metadata.
func WithUserID(ctx context.Context, userID string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, HeaderUserID, userID)
}
