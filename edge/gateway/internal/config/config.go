package config

import "os"

type Config struct {
	Port              string
	JWTSecret         string
	AuthGRPCAddr      string
	ProfileGRPCAddr   string
	MessagingGRPCAddr string
	DeliveryGRPCAddr  string
	ServiceName       string
	MetricsEnabled    bool
	TracingEnabled    bool
	JaegerURL         string
	HTTPAddr          string
}

func Load() *Config {
	return &Config{
		Port:              mustEnv("PORT"),
		JWTSecret:         mustEnv("JWT_SECRET"),
		AuthGRPCAddr:      mustEnv("AUTH_GRPC_ADDR"),
		ProfileGRPCAddr:   mustEnv("PROFILE_GRPC_ADDR"),
		MessagingGRPCAddr: mustEnv("MSG_GRPC_ADDR"),
		DeliveryGRPCAddr:  mustEnv("DELIVERY_GRPC_ADDR"),
		ServiceName:       mustEnv("SERVICE_NAME"),
		MetricsEnabled:    getEnvBool("METRICS_ENABLED", false),
		TracingEnabled:    getEnvBool("TRACING_ENABLED", false),
		JaegerURL:         mustEnv("JAEGER_URL"),
		HTTPAddr:          getEnv("HTTP_ADDR", ":8081"),
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("missing required environment variable: " + key)
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
