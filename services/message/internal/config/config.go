package config

import (
	"os"
)

type Config struct {
	GRPCAddr            string
	DatabaseURL         string
	KafkaBrokers        string
	KafkaTopic          string
	RedisAddr           string
	ServiceName         string
	HTTPAddr            string
	MetricsEnabled      bool
	TracingEnabled      bool
	JaegerURL           string
	ConversationSvcAddr string
}

func Load() *Config {
	return &Config{
		GRPCAddr:            mustEnv("GRPC_ADDR"),
		DatabaseURL:         mustEnv("DATABASE_URL"),
		KafkaBrokers:        mustEnv("KAFKA_BROKERS"),
		KafkaTopic:          mustEnv("KAFKA_TOPIC"),
		RedisAddr:           mustEnv("REDIS_ADDR"),
		ServiceName:         mustEnv("SERVICE_NAME"),
		HTTPAddr:            mustEnv("HTTP_ADDR"),
		MetricsEnabled:      getEnvBool("METRICS_ENABLED", false),
		TracingEnabled:      getEnvBool("TRACING_ENABLED", false),
		JaegerURL:           mustEnv("JAEGER_URL"),
		ConversationSvcAddr: getEnv("CONVERSATION_SVC_ADDR", "localhost:50055"),
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
		panic("missing required env: " + k)
	}
	return v
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
