package transport

import (
	"context"

	"google.golang.org/grpc/metadata"
)

const (
	HeaderUserID    = "x-user-id"
	HeaderRequestID = "x-request-id"
)

// WithUserID injects the authenticated user's ID into outgoing gRPC metadata.
func WithUserID(ctx context.Context, userID string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, HeaderUserID, userID)
}

// WithRequestID propagates the request-ID trace header to downstream gRPC calls.
func WithRequestID(ctx context.Context, reqID string) context.Context {
	if reqID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, HeaderRequestID, reqID)
}

// WithMeta is a convenience wrapper that injects both user-ID and request-ID
// in a single call — the common case for authenticated handler calls.
func WithMeta(ctx context.Context, userID, reqID string) context.Context {
	return WithRequestID(WithUserID(ctx, userID), reqID)
}
