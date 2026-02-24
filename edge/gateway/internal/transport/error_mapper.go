package transport

import (
	"log/slog"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		slog.Error("internal_error", "error", err)
		WriteError(w, 500, "internal_error", "an unexpected error occurred")
		return
	}

	// Always log the actual gRPC error on the server
	slog.Warn("grpc_error", "code", st.Code().String(), "message", st.Message())

	switch st.Code() {
	case codes.NotFound:
		WriteError(w, 404, "not_found", st.Message())
	case codes.InvalidArgument:
		WriteError(w, 400, "invalid_argument", st.Message())
	case codes.Unauthenticated:
		WriteError(w, 401, "unauthorized", "authentication failed")
	case codes.PermissionDenied:
		WriteError(w, 403, "forbidden", "access denied")
	case codes.AlreadyExists:
		WriteError(w, 409, "already_exists", st.Message())
	case codes.Unavailable:
		WriteError(w, 503, "unavailable", "service temporarily unavailable")
	case codes.DeadlineExceeded:
		WriteError(w, 504, "timeout", "request timed out")
	default:
		WriteError(w, 500, "internal_error", "an unexpected error occurred")
	}
}
