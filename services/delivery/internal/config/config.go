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
		ReqHTTPAddr:         fixPort(mustEnv("HTTP_PORT")),
		RedisAddr:           mustEnv("REDIS_ADDR"),
		KafkaBrokers:        strings.Split(mustEnv("KAFKA_BROKERS"), ","),
		KafkaTopics:         strings.Split(mustEnv("KAFKA_TOPICS"), ","),
		InstanceID:          mustEnv("INSTANCE_ID"),
		MessagingSvcAddr:    mustEnv("MSG_SVC_ADDR"),
		ConversationSvcAddr: mustEnv("CONV_SVC_ADDR"),
		PresenceSvcAddr:     mustEnv("PRESENCE_SVC_ADDR"),
		JWTSecret:           mustEnv("JWT_SECRET"),
		ServiceName:         mustEnv("SERVICE_NAME"),
		MetricsEnabled:      getEnvBool("METRICS_ENABLED", false),
		TracingEnabled:      getEnvBool("TRACING_ENABLED", false),
		JaegerURL:           getEnv("JAEGER_URL", "http://jaeger:14268/api/traces"),
		ObsHTTPAddr:         fixPort(mustEnv("HTTP_ADDR")),
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
