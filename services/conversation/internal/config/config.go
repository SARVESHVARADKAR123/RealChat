package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	GRPCAddr       string
	DatabaseURL    string
	KafkaBrokers   string
	KafkaTopic     string
	RedisAddr      string
	ServiceName    string
	ObsHTTPAddr    string
	MetricsEnabled bool
	TracingEnabled bool
	JaegerURL      string
}

func Load() *Config {
	return &Config{
		GRPCAddr:       fixPort(mustEnv("GRPC_ADDR")),
		DatabaseURL:    mustEnv("DATABASE_URL"),
		KafkaBrokers:   mustEnv("KAFKA_BROKERS"),
		KafkaTopic:     mustEnv("KAFKA_TOPIC"),
		RedisAddr:      mustEnv("REDIS_ADDR"),
		ServiceName:    mustEnv("SERVICE_NAME"),
		ObsHTTPAddr:    fixPort(mustEnv("HTTP_ADDR")),
		MetricsEnabled: getEnvBool("METRICS_ENABLED", false),
		TracingEnabled: getEnvBool("TRACING_ENABLED", false),
		JaegerURL:      getEnv("JAEGER_URL", "http://jaeger:14268/api/traces"),
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

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
