package router

import (
	"net/http"

	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/config"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/handlers"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/middleware"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/observability"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func NewRouter(
	authH *handlers.AuthHandler,
	profileH *handlers.ProfileHandler,
	convH *handlers.ConversationHandler,
	msgH *handlers.MessageHandler,
	presenceH *handlers.PresenceHandler,
	cfg *config.Config,
) http.Handler {

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(observability.MetricsMiddleware(cfg.ServiceName))
	r.Use(middleware.Recovery())

	r.Post("/api/login", authH.Login)
	r.Post("/api/register", authH.Register)
	r.Post("/api/refresh", authH.Refresh)
	r.Post("/api/logout", authH.Logout)

	r.Group(func(p chi.Router) {
		p.Use(middleware.JWT(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience))

		profilePath := "/api/profile"
		p.Get(profilePath, profileH.GetProfile)
		p.Patch(profilePath, profileH.UpdateProfile)

		convPath := "/api/conversations"
		p.Post(convPath, convH.CreateConversation)
		p.Get(convPath, convH.ListConversations)
		p.Get(convPath+"/{id}", convH.GetConversation)

		mesPath := "/api/messages"
		p.Get(mesPath, msgH.SyncMessages)
		p.Post(mesPath, msgH.SendMessage)
		p.Delete(mesPath, msgH.DeleteMessage)

		partPath := "/api/participants"
		p.Post(partPath, convH.AddParticipant)
		p.Delete(partPath, convH.RemoveParticipant)

		receiptPath := "/api/read-receipt"
		p.Post(receiptPath, convH.ReadReceipt)

		presencePath := "/api/presence"
		p.Get(presencePath, presenceH.GetPresence)
	})

	return otelhttp.NewHandler(r, "gateway")
}
