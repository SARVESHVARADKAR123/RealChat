package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net"

	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/config"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/dispatcher"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/kafka"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/membership"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/observability"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/presence"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/presencewatcher"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/router"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/server"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	messagingv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/messaging/v1"
	presencev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/presence/v1"
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

	instanceID := getOrGenerateInstanceID(cfg.InstanceID)

	redisClient := initRedis(ctx, cfg.RedisAddr, log)
	rtr := router.New(redisClient, instanceID)
	pres := presence.New(cfg.RedisAddr, instanceID)
	reg := websocket.NewRegistry()
	memCache := membership.New()

	messagingClient, messagingConn := initMessagingClient(cfg.MessagingSvcAddr, log)
	defer messagingConn.Close()

	disp := dispatcher.New(reg, memCache, pres, rtr, instanceID, messagingClient)
	pw := presencewatcher.NewWatcher(redisClient, reg, memCache)
	pw.Start(ctx)

	rtr.Subscribe(ctx, disp.DeliverRemote)
	wsHandler := websocket.NewHandler(reg, pres, messagingClient)

	// Kafka Consumer
	consumer := initKafka(ctx, cfg, disp, log)
	defer consumer.Close()

	// Servers
	obsSrv := initObservabilityServer(cfg, log)
	wsSrv := server.New(":"+cfg.HTTPPort, initMainRouter(wsHandler))
	grpcSrv := initPresenceGRPC(cfg, pres, log)

	startServers(cfg, obsSrv, wsSrv, log)

	<-ctx.Done()
	performGracefulShutdown(obsSrv, wsSrv, grpcSrv, reg, log)
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

func getOrGenerateInstanceID(id string) string {
	if id == "" {
		return uuid.NewString()
	}
	return id
}

func initRedis(ctx context.Context, addr string, log *zap.Logger) *redis.Client {
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatal("failed to connect to redis", zap.Error(err))
	}
	return client
}

func initMessagingClient(addr string, log *zap.Logger) (messagingv1.MessagingApiClient, *grpc.ClientConn) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("failed to connect to messaging service", zap.Error(err))
	}
	return messagingv1.NewMessagingApiClient(conn), conn
}

func initKafka(ctx context.Context, cfg *config.Config, disp *dispatcher.Dispatcher, log *zap.Logger) *kafka.Consumer {
	consumer, err := kafka.New(cfg.KafkaBrokers, cfg.KafkaTopics, disp)
	if err != nil {
		log.Fatal("failed to create kafka consumer", zap.Error(err))
	}
	consumer.Start(ctx)
	return consumer
}

func initObservabilityServer(cfg *config.Config, log *zap.Logger) *http.Server {
	mux := chi.NewRouter()
	mux.Use(observability.MetricsMiddleware(cfg.ServiceName))
	mux.Handle("/metrics", promhttp.Handler())
	mux.Get("/health/live", observability.HealthLiveHandler)
	mux.Get("/health/ready", observability.HealthReadyHandler())
	return &http.Server{Addr: cfg.HTTPAddr, Handler: mux}
}

func initMainRouter(wsHandler *websocket.Handler) http.Handler {
	mux := chi.NewRouter()
	mux.Handle("/ws", wsHandler)
	return mux
}

func initPresenceGRPC(cfg *config.Config, pres *presence.Presence, log *zap.Logger) *grpc.Server {
	srv := grpc.NewServer()
	presencev1.RegisterPresenceApiServer(srv, presence.NewGRPCHandler(pres))
	go func() {
		log.Info("starting presence grpc server", zap.String("addr", cfg.GRPCAddr))
		lis, err := net.Listen("tcp", cfg.GRPCAddr)
		if err != nil {
			log.Fatal("failed to listen for grpc", zap.Error(err))
		}
		if err := srv.Serve(lis); err != nil {
			log.Error("presence grpc server error", zap.Error(err))
		}
	}()
	return srv
}

func startServers(cfg *config.Config, obsSrv *http.Server, wsSrv *server.Server, log *zap.Logger) {
	go func() {
		log.Info("starting observability server", zap.String("addr", cfg.HTTPAddr))
		if err := obsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("observability server error", zap.Error(err))
		}
	}()
	go func() {
		log.Info("starting main server", zap.String("port", cfg.HTTPPort))
		if err := wsSrv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()
}

func performGracefulShutdown(obs *http.Server, ws *server.Server, grpcSrv *grpc.Server, reg *websocket.Registry, log *zap.Logger) {
	log.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ws.Shutdown(ctx); err != nil {
		log.Error("error during main server shutdown", zap.Error(err))
	}
	if err := obs.Shutdown(ctx); err != nil {
		log.Error("error during observability server shutdown", zap.Error(err))
	}
	grpcSrv.GracefulStop()
	reg.CloseAll()
	log.Info("shutdown complete, exiting")
}
