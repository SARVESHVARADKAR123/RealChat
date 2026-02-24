package middleware

import (
	"net/http"
	"time"

	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/observability"
	"go.uber.org/zap"
)

func Logger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)

			log := observability.GetLogger(r.Context())
			log.Info("http_request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int64("duration_ms", time.Since(start).Milliseconds()),
				zap.String("request_id", RequestIDFromContext(r.Context())),
			)
		})
	}
}
