package config

import (
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
}

func Load() *Config {
	return &Config{
		Port:                 getEnv("PORT", "8080"),
		JWTSecret:            getEnv("JWT_SECRET", "secret"),
		AuthGRPCAddr:         getEnv("AUTH_GRPC_ADDR", "localhost:50051"),
		ProfileGRPCAddr:      getEnv("PROFILE_GRPC_ADDR", "localhost:50052"),
		MessagingGRPCAddr:    getEnv("MSG_GRPC_ADDR", "localhost:50053"),
		ConversationGRPCAddr: getEnv("CONV_GRPC_ADDR", "conversation:50055"),
		PresenceGRPCAddr:     getEnv("PRESENCE_GRPC_ADDR", "presence:50056"),
		ServiceName:          getEnv("SERVICE_NAME", "gateway"),
		JWTIssuer:            getEnv("JWT_ISSUER", "realchat-auth"),
		JWTAudience:          getEnv("JWT_AUDIENCE", "realchat-clients"),
		MetricsEnabled:       getEnvBool("METRICS_ENABLED", false),
		TracingEnabled:       getEnvBool("TRACING_ENABLED", false),
		JaegerURL:            getEnv("JAEGER_URL", "http://localhost:14268/api/traces"),
		ObsHTTPAddr:          fixPort(getEnv("HTTP_ADDR", ":8090")),
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
