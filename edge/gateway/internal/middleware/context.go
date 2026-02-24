package middleware

import "context"

type ctxKey int

const (
	userIDKey ctxKey = iota
	requestIDKey
)

func InjectUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func UserID(ctx context.Context) string {
	v := ctx.Value(userIDKey)
	if v == nil {
		return ""
	}
	return v.(string)
}

func RequestIDFromContext(ctx context.Context) string {
	v := ctx.Value(requestIDKey)
	if v == nil {
		return ""
	}
	return v.(string)
}
