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
		RedisAddr:      mustEnv("REDIS_ADDR"),
		GRPCAddr:       fixPort(mustEnv("GRPC_ADDR")),
		ObsHTTPAddr:    fixPort(mustEnv("HTTP_ADDR")),
		InstanceID:     mustEnv("INSTANCE_ID"),
		ServiceName:    mustEnv("SERVICE_NAME"),
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
