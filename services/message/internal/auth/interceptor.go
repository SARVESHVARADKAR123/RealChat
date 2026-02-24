package auth

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	HeaderUserID            = "x-user-id"
)

// Interceptor extracts the x-user-id header and injects it into the context.
func Interceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// If metadata is missing but it's GetConversation, we allow it (internal call)
		if info.FullMethod == "/messaging.v1.MessagingApi/GetConversation" {
			return handler(ctx, req)	
		}
		return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	values := md.Get(HeaderUserID)
	if len(values) == 0 || values[0] == "" {
		// Exempt GetConversation from mandatory header
		if info.FullMethod == "/messaging.v1.MessagingApi/GetConversation" {
			return handler(ctx, req)	
		}
		return nil, status.Error(codes.Unauthenticated, "x-user-id header is missing")
	}

	userID := values[0]
	newCtx := context.WithValue(ctx, UserIDKey, userID)

	return handler(newCtx, req)
}

// GetUserID retrieves the authenticated user ID from context.
func GetUserID(ctx context.Context) (string, error) {
	val := ctx.Value(UserIDKey)
	if val == nil {
		return "", errors.New("user id not found in context")
	}
	id, ok := val.(string)
	if !ok {
		return "", errors.New("invalid user id type")
	}
	return id, nil
}
