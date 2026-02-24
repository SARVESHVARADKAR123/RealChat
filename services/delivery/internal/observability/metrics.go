package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"service", "method", "path", "status"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "path"},
	)

	WebSocketConnectionsTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "websocket_connections_active",
			Help: "Current number of active WebSocket connections",
		},
		[]string{"service"},
	)

	KafkaConsumerLag = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_consumer_lag",
			Help: "Kafka consumer lag per partition",
		},
		[]string{"service", "topic", "partition"},
	)

	MessageDeliveryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "message_delivery_latency_seconds",
			Help:    "Latency of message delivery from Kafka to WebSocket",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service"},
	)
)
