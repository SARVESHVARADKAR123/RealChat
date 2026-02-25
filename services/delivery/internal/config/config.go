package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPPort            string
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
	HTTPAddr            string
}

func Load() *Config {
	return &Config{
		HTTPPort:            mustEnv("HTTP_PORT"),
		RedisAddr:           mustEnv("REDIS_ADDR"),
		KafkaBrokers:        strings.Split(mustEnv("KAFKA_BROKERS"), ","),
		KafkaTopics:         strings.Split(mustEnv("KAFKA_TOPICS"), ","),
		InstanceID:          getEnv("INSTANCE_ID", getEnv("HOSTNAME", "")),
		MessagingSvcAddr:    mustEnv("MSG_SVC_ADDR"),
		ConversationSvcAddr: mustEnv("CONV_SVC_ADDR"),
		PresenceSvcAddr:     mustEnv("PRESENCE_SVC_ADDR"),
		JWTSecret:           mustEnv("JWT_SECRET"),
		ServiceName:         mustEnv("SERVICE_NAME"),
		MetricsEnabled:      getEnvBool("METRICS_ENABLED", false),
		TracingEnabled:      getEnvBool("TRACING_ENABLED", false),
		JaegerURL:           mustEnv("JAEGER_URL"),
		HTTPAddr:            getEnv("HTTP_ADDR", ":8081"),
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
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
