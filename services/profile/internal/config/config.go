package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL    string
	RedisAddr      string
	KafkaBrokers   string
	JWTIssuer      string
	JWTAudience    string
	JWTSecret      string
	HTTPPort       string
	GRPCAddr       string
	ServiceName    string
	MetricsEnabled bool
	TracingEnabled bool
	JaegerURL      string
	ObsHTTPAddr    string
}

func Load() *Config {
	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/profile?sslmode=disable"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		KafkaBrokers:   getEnv("KAFKA_BROKERS", "localhost:9092"),
		JWTIssuer:      getEnv("JWT_ISSUER", "realchat"),
		JWTAudience:    getEnv("JWT_AUDIENCE", "realchat"),
		JWTSecret:      getEnv("JWT_SECRET", "secret"),
		HTTPPort:       fixPort(getEnv("HTTP_PORT", "8082")),
		GRPCAddr:       fixPort(getEnv("GRPC_ADDR", ":50051")),
		ServiceName:    getEnv("SERVICE_NAME", "profile-service"),
		MetricsEnabled: getEnvBool("METRICS_ENABLED", false),
		TracingEnabled: getEnvBool("TRACING_ENABLED", false),
		JaegerURL:      getEnv("JAEGER_URL", "http://localhost:14268/api/traces"),
		ObsHTTPAddr:    fixPort(getEnv("HTTP_ADDR", ":8092")),
	}
}

func fixPort(port string) string {
	if port != "" && !strings.Contains(port, ":") {
		return ":" + port
	}
	return port
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v == "true"
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("missing required env: %s", k)
	}
	return v
}

func getEnv(k, d string) string {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	return v
}
