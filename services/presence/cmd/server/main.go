package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	presencev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/presence/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/presence/internal/config"
	"github.com/SARVESHVARADKAR123/RealChat/services/presence/internal/observability"
	"github.com/SARVESHVARADKAR123/RealChat/services/presence/internal/presence"
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
		defer tp.Shutdown(context.Background())
	}

	ctx, cancel := setupSignalHandler(log)
	defer cancel()

	// Redis connectivity check
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("failed to connect to redis", zap.Error(err))
	}
	redisClient.Close() // Close check client; Presence creates its own

	// Presence core
	pres := presence.New(cfg.RedisAddr, cfg.InstanceID)

	// gRPC server
	grpcSrv := grpc.NewServer()
	presencev1.RegisterPresenceApiServer(grpcSrv, presence.NewGRPCHandler(pres))

	go func() {
		log.Info("starting presence grpc server", zap.String("addr", cfg.GRPCAddr))
		lis, err := net.Listen("tcp", cfg.GRPCAddr)
		if err != nil {
			log.Fatal("failed to listen for grpc", zap.Error(err))
		}
		if err := grpcSrv.Serve(lis); err != nil {
			log.Error("presence grpc server error", zap.Error(err))
		}
	}()

	// HTTP Observability server (metrics + health)
	obsMux := chi.NewRouter()
	obsMux.Use(observability.MetricsMiddleware(cfg.ServiceName))
	obsMux.Handle("/metrics", promhttp.Handler())
	obsMux.Get("/health/live", observability.HealthLiveHandler)
	obsMux.Get("/health/ready", observability.HealthReadyHandler())

	obsSrv := &http.Server{Addr: cfg.ObsHTTPAddr, Handler: obsMux}
	go func() {
		log.Info("starting observability server", zap.String("addr", cfg.ObsHTTPAddr))
		if err := obsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("observability server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	performGracefulShutdown(obsSrv, grpcSrv, redisClient, log)
}

func setupSignalHandler(log *zap.Logger) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Info("received signal, initiating shutdown", zap.String("signal", sig.String()))
		cancel()
	}()
	return ctx, cancel
}

func performGracefulShutdown(obs *http.Server, grpcSrv *grpc.Server, redisClient *redis.Client, log *zap.Logger) {
	log.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := obs.Shutdown(ctx); err != nil {
		log.Error("error during observability server shutdown", zap.Error(err))
	}
	grpcSrv.GracefulStop()
	if redisClient != nil {
		redisClient.Close()
	}
	log.Info("shutdown complete, exiting")
}
