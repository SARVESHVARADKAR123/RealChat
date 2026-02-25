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
		return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	// Extract User ID
	var newCtx context.Context = ctx
	userValues := md.Get(HeaderUserID)
	if len(userValues) > 0 && userValues[0] != "" {
		newCtx = context.WithValue(newCtx, UserIDKey, userValues[0])
	} else {
		return nil, status.Error(codes.Unauthenticated, "x-user-id header is missing")
	}

	// Extract Request ID
	reqValues := md.Get(HeaderRequestID)
	if len(reqValues) > 0 && reqValues[0] != "" {
		newCtx = context.WithValue(newCtx, RequestIDKey, reqValues[0])
	}

	return handler(newCtx, req)
}

// ClientInterceptor propagates user-id and request-id to outgoing gRPC calls.
func ClientInterceptor(
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	// Propagate UserID
	if userID, err := GetUserID(ctx); err == nil {
		ctx = metadata.AppendToOutgoingContext(ctx, HeaderUserID, userID)
	}

	// Propagate RequestID
	if val := ctx.Value(RequestIDKey); val != nil {
		if reqID, ok := val.(string); ok {
			ctx = metadata.AppendToOutgoingContext(ctx, HeaderRequestID, reqID)
		}
	}

	return invoker(ctx, method, req, reply, cc, opts...)
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
