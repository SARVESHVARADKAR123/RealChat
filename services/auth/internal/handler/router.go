package handler

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/middleware"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/observability"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/service"
)

// NewRouter builds the HTTP router with all auth routes.
func NewRouter(svc *service.AuthService, db *sql.DB) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Timeout(15 * time.Second))
	r.Use(observability.MetricsMiddleware("auth"))

	h := NewAuthHandler(svc, db)

	// Auth routes
	path := "/api/v1/auth"
	r.Post(path+"/register", h.Register)
	r.Post(path+"/login", h.Login)
	r.Post(path+"/refresh", h.Refresh)
	r.Post(path+"/logout", h.Logout)

	return r
}
