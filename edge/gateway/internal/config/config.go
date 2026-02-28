package config

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type Config struct {
	Port                 string
	JWTSecret            string
	AuthGRPCAddr         string
	ProfileGRPCAddr      string
	MessagingGRPCAddr    string
	ConversationGRPCAddr string
	PresenceGRPCAddr     string
	ServiceName          string
	JWTIssuer            string
	JWTAudience          string
	MetricsEnabled       bool
	TracingEnabled       bool
	JaegerURL            string
	ObsHTTPAddr          string

	// Rate Limiting
	RateLimitRequests int
	RateLimitWindow   string
}

func Load() *Config {
	return &Config{
		Port:                 mustEnv("PORT"),
		JWTSecret:            mustEnv("JWT_SECRET"),
		AuthGRPCAddr:         mustEnv("AUTH_GRPC_ADDR"),
		ProfileGRPCAddr:      mustEnv("PROFILE_GRPC_ADDR"),
		MessagingGRPCAddr:    mustEnv("MSG_GRPC_ADDR"),
		ConversationGRPCAddr: mustEnv("CONV_GRPC_ADDR"),
		PresenceGRPCAddr:     mustEnv("PRESENCE_GRPC_ADDR"),
		ServiceName:          mustEnv("SERVICE_NAME"),
		JWTIssuer:            getEnv("JWT_ISSUER", "realchat-auth"), // Keep sensible defaults for internal constants
		JWTAudience:          getEnv("JWT_AUDIENCE", "realchat-clients"),
		MetricsEnabled:       getEnvBool("METRICS_ENABLED", false),
		TracingEnabled:       getEnvBool("TRACING_ENABLED", false),
		JaegerURL:            getEnv("JAEGER_URL", "http://jaeger:14268/api/traces"),
		ObsHTTPAddr:          fixPort(mustEnv("HTTP_ADDR")),
		RateLimitRequests:    getEnvInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:      getEnv("RATE_LIMIT_WINDOW", "1m"),
	}
}

func fixPort(port string) string {
	if port != "" && !strings.Contains(port, ":") {
		return ":" + port
	}
	return port
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env: %s", key)
	}
	return v
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v == "true"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var i int
	if _, err := fmt.Sscanf(v, "%d", &i); err != nil {
		return fallback
	}
	return i
}
