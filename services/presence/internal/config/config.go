package config

import (
	"os"
)

type Config struct {
	RedisAddr      string
	GRPCAddr       string
	HTTPAddr       string
	InstanceID     string
	ServiceName    string
	MetricsEnabled bool
	TracingEnabled bool
	JaegerURL      string
}

func Load() *Config {
	return &Config{
		RedisAddr:      mustEnv("REDIS_ADDR"),
		GRPCAddr:       getEnv("GRPC_ADDR", ":50056"),
		HTTPAddr:       getEnv("HTTP_ADDR", ":8096"),
		InstanceID:     getEnv("INSTANCE_ID", ""),
		ServiceName:    mustEnv("SERVICE_NAME"),
		MetricsEnabled: getEnvBool("METRICS_ENABLED", false),
		TracingEnabled: getEnvBool("TRACING_ENABLED", false),
		JaegerURL:      mustEnv("JAEGER_URL"),
	}
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		panic("missing required env: " + k)
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
