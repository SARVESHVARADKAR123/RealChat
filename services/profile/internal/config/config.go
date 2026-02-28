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
		DatabaseURL:    mustEnv("DATABASE_URL"),
		RedisAddr:      mustEnv("REDIS_ADDR"),
		KafkaBrokers:   mustEnv("KAFKA_BROKERS"),
		JWTIssuer:      getEnv("JWT_ISSUER", "realchat-auth"), // Keep sensible defaults for internal constants
		JWTAudience:    getEnv("JWT_AUDIENCE", "realchat-clients"),
		JWTSecret:      mustEnv("JWT_SECRET"),
		HTTPPort:       fixPort(mustEnv("HTTP_PORT")),
		GRPCAddr:       fixPort(mustEnv("GRPC_ADDR")),
		ServiceName:    mustEnv("SERVICE_NAME"),
		MetricsEnabled: getEnvBool("METRICS_ENABLED", false),
		TracingEnabled: getEnvBool("TRACING_ENABLED", false),
		JaegerURL:      getEnv("JAEGER_URL", "http://jaeger:14268/api/traces"),
		ObsHTTPAddr:    fixPort(mustEnv("HTTP_ADDR")),
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
