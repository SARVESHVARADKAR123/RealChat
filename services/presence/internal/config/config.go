package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	RedisAddr      string
	GRPCAddr       string
	ObsHTTPAddr    string
	InstanceID     string
	ServiceName    string
	MetricsEnabled bool
	TracingEnabled bool
	JaegerURL      string
}

func Load() *Config {
	return &Config{
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		GRPCAddr:       fixPort(getEnv("GRPC_ADDR", ":50056")),
		ObsHTTPAddr:    fixPort(getEnv("HTTP_ADDR", ":8096")),
		InstanceID:     getEnv("INSTANCE_ID", ""),
		ServiceName:    getEnv("SERVICE_NAME", "presence-service"),
		MetricsEnabled: getEnvBool("METRICS_ENABLED", false),
		TracingEnabled: getEnvBool("TRACING_ENABLED", false),
		JaegerURL:      getEnv("JAEGER_URL", "http://localhost:14268/api/traces"),
	}
}

func fixPort(port string) string {
	if port != "" && !strings.Contains(port, ":") {
		return ":" + port
	}
	return port
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("missing required env: %s", k)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v == "true"
}
