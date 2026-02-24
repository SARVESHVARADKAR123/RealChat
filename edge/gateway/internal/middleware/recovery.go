package middleware

import (
	"net/http"

	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/observability"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/transport"
	"go.uber.org/zap"
)

func Recovery() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			defer func() {
				if rec := recover(); rec != nil {
					log := observability.GetLogger(r.Context())
					log.Error("panic_recovered",
						zap.Any("error", rec),
						zap.String("request_id", RequestIDFromContext(r.Context())),
					)

					transport.WriteError(
						w,
						http.StatusInternalServerError,
						"internal_error",
						"internal server error",
					)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
