package handler

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/config"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/middleware"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/service"
)

// NewRouter builds the HTTP router with all profile routes.
func NewRouter(cfg *config.Config, p *service.ProfileService, c *service.ContactService, b *service.BlockService, db *sql.DB) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Timeout(15 * time.Second))

	// Auth middleware
	auth := middleware.JWT([]byte(cfg.JWTSecret), cfg.JWTIssuer, cfg.JWTAudience)

	ph := NewProfileHandler(p)
	ch := NewContactHandler(c)
	bh := NewBlockHandler(b)

	// Profile routes
	path := "/api/v1/profile"
	r.With(auth).Get(path+"/me", ph.Get)
	r.With(auth).Put(path+"/me", ph.Update)

	// Contact routes
	contactPath := path + "/contacts"
	r.With(auth).Post(contactPath, ch.Add)
	r.With(auth).Delete(contactPath, ch.Remove)
	r.With(auth).Get(contactPath, ch.List)

	// Block routes
	blockPath := path + "/blocks"
	r.With(auth).Post(blockPath, bh.Block)
	r.With(auth).Delete(blockPath, bh.Unblock)

	// Health
	healthPath := "/health"
	r.Get(healthPath, Health())
	r.Get(healthPath+"/ready", Ready(db))

	return r
}
