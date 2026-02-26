package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	ReqHTTPAddr         string
	RedisAddr           string
	KafkaBrokers        []string
	KafkaTopics         []string
	InstanceID          string
	MessagingSvcAddr    string
	ConversationSvcAddr string
	PresenceSvcAddr     string
	JWTSecret           string
	ServiceName         string
	MetricsEnabled      bool
	TracingEnabled      bool
	JaegerURL           string
	ObsHTTPAddr         string
}

func Load() *Config {
	return &Config{
		ReqHTTPAddr:         fixPort(getEnv("HTTP_PORT", ":8083")),
		RedisAddr:           getEnv("REDIS_ADDR", "localhost:6379"),
		KafkaBrokers:        strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaTopics:         strings.Split(getEnv("KAFKA_TOPICS", "message-events"), ","),
		InstanceID:          getEnv("INSTANCE_ID", getEnv("HOSTNAME", "")),
		MessagingSvcAddr:    getEnv("MSG_SVC_ADDR", "localhost:50052"),
		ConversationSvcAddr: getEnv("CONV_SVC_ADDR", "localhost:50051"),
		PresenceSvcAddr:     getEnv("PRESENCE_SVC_ADDR", "localhost:50053"),
		JWTSecret:           getEnv("JWT_SECRET", "secret"),
		ServiceName:         getEnv("SERVICE_NAME", "delivery-service"),
		MetricsEnabled:      getEnvBool("METRICS_ENABLED", false),
		TracingEnabled:      getEnvBool("TRACING_ENABLED", false),
		JaegerURL:           getEnv("JAEGER_URL", "http://localhost:14268/api/traces"),
		ObsHTTPAddr:         fixPort(getEnv("HTTP_ADDR", ":8093")),
	}
}

func fixPort(port string) string {
	if port != "" && !strings.HasPrefix(port, ":") {
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
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
