package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

// RateLimit returns a middleware that limits requests by IP address
// based on the configured limits and window duration.
func RateLimit(requests int, windowStr string) func(next http.Handler) http.Handler {
	window, err := time.ParseDuration(windowStr)
	if err != nil {
		window = time.Minute // Default fallback
	}

	return httprate.LimitByIP(requests, window)
}
