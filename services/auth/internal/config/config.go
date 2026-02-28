package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// ───── Infrastructure ─────
	DatabaseURL  string
	RedisAddr    string
	KafkaBrokers []string

	// ───── Runtime ─────
	HTTPAddr    string
	GRPCAddr    string
	ObsHTTPAddr string
	ServiceName string
	LogLevel    string

	// ───── JWT Security ─────
	JWTSecret   string
	JWTIssuer   string
	JWTAudience string

	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// ───── Rate Limiting ─────
	LoginRateLimitPerMin   int
	RefreshRateLimitPerMin int

	// ───── Observability ─────
	MetricsEnabled bool
	TracingEnabled bool
	JaegerURL      string
}

func Load() Config {
	return Config{
		// Infra
		DatabaseURL:  mustEnv("DATABASE_URL"),
		RedisAddr:    mustEnv("REDIS_ADDR"),
		KafkaBrokers: getEnvSlice("KAFKA_BROKERS", nil),

		// Runtime
		HTTPAddr:    fixPort(mustEnv("HTTP_ADDR")),
		GRPCAddr:    fixPort(mustEnv("GRPC_ADDR")),
		ObsHTTPAddr: fixPort(mustEnv("OBS_HTTP_ADDR")),
		ServiceName: mustEnv("SERVICE_NAME"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		// JWT
		JWTSecret:   mustEnv("JWT_SECRET"),
		JWTIssuer:   getEnv("JWT_ISSUER", "realchat-auth"),
		JWTAudience: getEnv("JWT_AUDIENCE", "realchat-clients"),

		AccessTokenTTL:  time.Duration(getEnvInt("ACCESS_TTL_MIN", 60)) * time.Minute,
		RefreshTokenTTL: time.Duration(getEnvInt("REFRESH_TTL_HOURS", 24)) * time.Hour,

		// Rate limiting
		LoginRateLimitPerMin:   getEnvInt("LOGIN_RATE_LIMIT", 10),
		RefreshRateLimitPerMin: getEnvInt("REFRESH_RATE_LIMIT", 20),

		// Observability
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

func getEnv(k, d string) string {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	return v
}

func getEnvInt(k string, d int) int {
	v := os.Getenv(k)
	if v == "" {
		return d
	}

	i, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("invalid int env %s: %v", k, err)
	}
	return i
}

func getEnvBool(k string, d bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	return strings.ToLower(v) == "true"
}

func getEnvSlice(k string, d []string) []string {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	return strings.Split(v, ",")
}
