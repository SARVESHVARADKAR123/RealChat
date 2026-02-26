package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/config"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/handler"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/kafka"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/observability"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/outbox"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/repository"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/service"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/transport/grpc"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/tx"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	// Database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal("db open failed", zap.Error(err))
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		log.Fatal("db ping failed", zap.Error(err))
	}

	// Kafka producer
	producer := kafka.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	// Wire dependencies
	txMgr := &tx.Manager{DB: db}
	repo := repository.NewAuthRepository(db)
	svc := service.NewAuthService(repo, cfg, producer, txMgr)

	// Outbox worker
	outboxWorker := outbox.NewWorker(db, producer, 100, 500*time.Millisecond)

	// HTTP server with chi router
	mux := handler.NewRouter(svc, db)
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}

	// gRPC server
	grpcSrv := grpc.NewServer(svc)

	go func() {
		log.Info("HTTP started", zap.String("service", cfg.ServiceName), zap.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	go func() {
		if err := grpcSrv.Start(cfg.GRPCAddr); err != nil {
			log.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	// HTTP Observability server
	obsMux := chi.NewRouter()
	obsMux.Use(observability.MetricsMiddleware("auth"))
	obsMux.Handle("/metrics", promhttp.Handler())
	obsMux.Get("/health/live", observability.HealthLiveHandler)
	obsMux.Get("/health/ready", observability.HealthReadyHandler(db))

	obsSrv := &http.Server{Addr: cfg.ObsHTTPAddr, Handler: obsMux}
	go func() {
		log.Info("Observability HTTP started", zap.String("addr", cfg.ObsHTTPAddr))
		if err := obsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Observability server failed", zap.Error(err))
		}
	}()

	// Context for background workers
	workerCtx, workerCancel := context.WithCancel(context.Background())
	go outboxWorker.Start(workerCtx)

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Info("shutting down...")

	workerCancel() // Stop outbox worker

	ctxShut, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = srv.Shutdown(ctxShut)
	_ = obsSrv.Shutdown(ctxShut)
	grpcSrv.Stop()
	log.Info("stopped", zap.String("service", cfg.ServiceName))
}
