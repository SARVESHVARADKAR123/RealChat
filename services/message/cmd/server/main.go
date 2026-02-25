package main

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	conversationv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/conversation/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/application"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/auth"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/cache"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/config"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/kafka"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/observability"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/outbox"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/repository/postgres"
	grpc_transport "github.com/SARVESHVARADKAR123/RealChat/services/message/internal/transport/grpc"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/tx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {

	cfg := config.Load()

	// Observability
	observability.InitLogger(cfg.ServiceName)
	log := observability.Log

	if cfg.TracingEnabled {
		tp, err := observability.InitTracer(cfg.ServiceName, cfg.JaegerURL)
		if err != nil {
			log.Fatal("failed to initialize tracer", zap.Error(err))
		}
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Error("failed to shutdown tracer provider", zap.Error(err))
			}
		}()
	}

	// HTTP Server for Observability (Metrics & Health)
	mux := chi.NewRouter()
	mux.Use(observability.MetricsMiddleware(cfg.ServiceName))
	mux.Handle("/metrics", promhttp.Handler())
	mux.Get("/health/live", observability.HealthLiveHandler)
	mux.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		// Messaging ready check
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	go func() {
		log.Info("HTTP observability server started", zap.String("addr", cfg.HTTPAddr))
		if err := http.ListenAndServe(cfg.HTTPAddr, mux); err != nil {
			log.Error("HTTP observability server failed", zap.Error(err))
		}
	}()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal("db open failed", zap.Error(err))
	}
	defer db.Close()

	// Redis Cache
	cacheClient := cache.New(cfg.RedisAddr)
	defer cacheClient.Client.Close()

	// gRPC Client to Conversation Service
	conn, err := grpc.Dial(
		cfg.ConversationSvcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(auth.ClientInterceptor),
	)
	if err != nil {
		log.Fatal("failed to connect to conversation service", zap.Error(err))
	}
	defer conn.Close()
	convSvcClient := conversationv1.NewConversationApiClient(conn)

	repo := &postgres.Repository{
		DB:    db,
		Cache: cacheClient,
	}
	txMgr := &tx.Manager{DB: db}
	app := application.New(repo, txMgr, convSvcClient, log)

	// Kafka Producer
	producer, err := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	if err != nil {
		log.Fatal("kafka producer failed", zap.Error(err))
	}

	// Outbox Worker
	worker := &outbox.Worker{
		DB:        db,
		Producer:  producer,
		BatchSize: 100,
		PollDelay: 2 * time.Second,
	}

	// Cancellable context for background workers
	ctx, cancel := context.WithCancel(context.Background())

	go worker.Start(ctx)

	// gRPC Server — blocks until SIGINT/SIGTERM
	server := grpc_transport.New(app)
	server.Start(cfg.GRPCAddr)

	// Shutdown: stop outbox worker and flush Kafka producer
	cancel()
	producer.Flush(5000) // wait up to 5s for any in-flight messages
	log.Info("shutdown complete")
}
