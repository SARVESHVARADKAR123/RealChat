package config

import (
	"log"
	"os"
)

type Config struct {
	DATABASE_URL   string
	RedisAddr      string
	KafkaBrokers   string
	JWTIssuer      string
	JWTAudience    string
	JWTSecret      string
	HTTPPort       string
	GRPC_ADDR      string
	ServiceName    string
	MetricsEnabled bool
	TracingEnabled bool
	JaegerURL      string
	HTTPAddr       string
}

func Load() *Config {
	return &Config{
		DATABASE_URL:   mustEnv("DATABASE_URL"),
		RedisAddr:      mustEnv("REDIS_ADDR"),
		KafkaBrokers:   mustEnv("KAFKA_BROKERS"),
		JWTIssuer:      mustEnv("JWT_ISSUER"),
		JWTAudience:    mustEnv("JWT_AUDIENCE"),
		JWTSecret:      mustEnv("JWT_SECRET"),
		HTTPPort:       mustEnv("HTTP_PORT"),
		GRPC_ADDR:      mustEnv("GRPC_ADDR"),
		ServiceName:    getEnv("SERVICE_NAME", "profile-service"),
		MetricsEnabled: getEnvBool("METRICS_ENABLED", false),
		TracingEnabled: getEnvBool("TRACING_ENABLED", false),
		JaegerURL:      getEnv("JAEGER_URL", "http://localhost:14268/api/traces"),
		HTTPAddr:       getEnv("HTTP_ADDR", ":8081"),
	}
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
