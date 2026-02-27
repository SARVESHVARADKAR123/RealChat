package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/application"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/cache"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/config"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/kafka"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/observability"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/outbox"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/repository/postgres"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/transport/grpc"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/tx"
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

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal("db open failed", zap.Error(err))
	}
	defer db.Close()

	// HTTP Server for Observability (Metrics & Health)
	mux := chi.NewRouter()
	mux.Use(observability.MetricsMiddleware(cfg.ServiceName))
	mux.Handle("/metrics", promhttp.Handler())
	mux.Get("/health/live", observability.HealthLiveHandler)
	mux.Get("/health/ready", observability.HealthReadyHandler(db))

	obsSrv := &http.Server{Addr: cfg.ObsHTTPAddr, Handler: mux}

	go func() {
		log.Info("HTTP observability server started", zap.String("addr", cfg.ObsHTTPAddr))
		if err := obsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP observability server failed", zap.Error(err))
		}
	}()

	// Redis Cache
	cacheClient := cache.New(cfg.RedisAddr)
	defer cacheClient.Client.Close()

	repo := &postgres.Repository{
		DB:    db,
		Cache: cacheClient,
	}
	txMgr := &tx.Manager{DB: db}
	app := application.New(repo, txMgr)

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
	defer cancel()

	go worker.Start(ctx)

	// gRPC Server
	server := grpc.New(app)
	go server.Start(cfg.GRPCAddr)

	// Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info("shutting down...")
	cancel()

	// HTTP Graceful shutdown
	ctxShut, obsCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer obsCancel()
	_ = obsSrv.Shutdown(ctxShut)

	// Give components time to finish
	producer.Flush(5000)
	server.Stop()

	log.Info("shutdown complete")
}
