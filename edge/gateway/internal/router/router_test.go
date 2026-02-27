package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/config"
)

func TestRateLimiting(t *testing.T) {
	// 1. Minimum config to trigger the rate limiter
	cfg := &config.Config{
		RateLimitRequests: 10,   // strictly 10 requests
		RateLimitWindow:   "1m", // per minute
	}

	// 2. We use an empty stand-in router to test middleware

	// 3. Instead of initializing all handlers, just plug in the bare middleware to
	// a mock endpoint
	handler := NewRouter(nil, nil, nil, nil, nil, cfg)

	server := httptest.NewServer(handler)
	defer server.Close()

	client := server.Client()

	// 4. Send 10 allowed requests
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", server.URL+"/api/presence", nil) // /api/presence requires JWT in real router, but rate limit triggers first
		req.Header.Set("X-Forwarded-For", "192.168.1.100")                // Simulate single IP
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed request %d: %v", i, err)
		}
		// Since we didn't mock JWT/Handler it'll be 401 or panic, but NOT 429
		if res.StatusCode == http.StatusTooManyRequests {
			t.Fatalf("Request %d got 429 too early", i)
		}
		res.Body.Close()
	}

	// 5. the 11th request MUST be rate limited (429)
	req, _ := http.NewRequest("GET", server.URL+"/api/presence", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	res, err := client.Do(req)

	if err != nil {
		t.Fatalf("Failed 11th request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status 429 Too Many Requests, got %d", res.StatusCode)
	}
}
