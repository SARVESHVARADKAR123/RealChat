package router

import (
	"net/http"

	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/handlers"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/middleware"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/observability"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func NewRouter(
	authH *handlers.AuthHandler,
	profileH *handlers.ProfileHandler,
	msgH *handlers.MessagingHandler,
	receiptH *handlers.ReceiptHandler,
	presenceH *handlers.PresenceHandler,
	secret string,
	serviceName string,
) http.Handler {

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(observability.MetricsMiddleware(serviceName))
	r.Use(middleware.Recovery())

	r.Get("/health/live", observability.HealthLiveHandler)
	r.Get("/health/ready", observability.HealthReadyHandler())

	r.Post("/api/login", authH.Login)
	r.Post("/api/register", authH.Register)

	r.Group(func(p chi.Router) {
		p.Use(middleware.JWT(secret))

		profilePath := "/api/profile"
		p.Get(profilePath, profileH.GetProfile)
		p.Patch(profilePath, profileH.UpdateProfile)

		convPath := "/api/conversations"
		p.Post(convPath, msgH.CreateConversation)
		p.Get(convPath, msgH.ListConversations)

		mesPath := "/api/messages"
		p.Get(mesPath, msgH.SyncMessages)
		p.Post(mesPath, msgH.SendMessage)
		p.Delete(mesPath, msgH.DeleteMessage)

		partPath := "/api/participants"
		p.Post(partPath, msgH.AddParticipant)
		p.Delete(partPath, msgH.RemoveParticipant)

		receiptPath := "/api/read-receipt"
		p.Post(receiptPath, receiptH.ReadReceipt)

		presencePath := "/api/presence"
		p.Get(presencePath, presenceH.GetPresence)
	})

	return otelhttp.NewHandler(r, "gateway")
}
