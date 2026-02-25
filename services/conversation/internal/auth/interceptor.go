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
	UserIDKey       contextKey = "user_id"
	RequestIDKey    contextKey = "request_id"
	HeaderUserID               = "x-user-id"
	HeaderRequestID            = "x-request-id"
)

// Interceptor extracts the x-user-id and x-request-id headers and injects them into the context.
func Interceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// If metadata is missing but it's GetConversation or NextSequence, we allow it (internal call)
		if info.FullMethod == "/realchat.conversation.v1.ConversationApi/GetConversation" ||
			info.FullMethod == "/realchat.conversation.v1.ConversationApi/NextSequence" {
			return handler(ctx, req)
		}
		return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	// Extract User ID
	var newCtx context.Context = ctx
	userValues := md.Get(HeaderUserID)
	if len(userValues) > 0 && userValues[0] != "" {
		newCtx = context.WithValue(newCtx, UserIDKey, userValues[0])
	} else {
		// Exempt GetConversation and NextSequence from mandatory user ID
		if info.FullMethod == "/realchat.conversation.v1.ConversationApi/GetConversation" ||
			info.FullMethod == "/realchat.conversation.v1.ConversationApi/NextSequence" {
			return handler(ctx, req)
		}
		return nil, status.Error(codes.Unauthenticated, "x-user-id header is missing")
	}

	// Extract Request ID
	reqValues := md.Get(HeaderRequestID)
	if len(reqValues) > 0 && reqValues[0] != "" {
		newCtx = context.WithValue(newCtx, RequestIDKey, reqValues[0])
	}

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
