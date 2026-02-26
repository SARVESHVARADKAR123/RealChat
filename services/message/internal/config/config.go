package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	GRPCAddr            string
	DatabaseURL         string
	KafkaBrokers        string
	KafkaTopic          string
	RedisAddr           string
	ServiceName         string
	ObsHTTPAddr         string
	MetricsEnabled      bool
	TracingEnabled      bool
	JaegerURL           string
	ConversationSvcAddr string
}

func Load() *Config {
	return &Config{
		GRPCAddr:            fixPort(getEnv("GRPC_ADDR", ":50052")),
		DatabaseURL:         mustEnv("DATABASE_URL"),
		KafkaBrokers:        getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:          getEnv("KAFKA_TOPIC", "message-events"),
		RedisAddr:           getEnv("REDIS_ADDR", "localhost:6379"),
		ServiceName:         getEnv("SERVICE_NAME", "message-service"),
		ObsHTTPAddr:         fixPort(getEnv("HTTP_ADDR", ":8082")),
		MetricsEnabled:      getEnvBool("METRICS_ENABLED", false),
		TracingEnabled:      getEnvBool("TRACING_ENABLED", false),
		JaegerURL:           getEnv("JAEGER_URL", "http://localhost:14268/api/traces"),
		ConversationSvcAddr: getEnv("CONVERSATION_SVC_ADDR", "localhost:50051"),
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
