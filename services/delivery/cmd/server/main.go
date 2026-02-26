package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/config"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/dispatcher"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/kafka"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/membership"
	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/observability"
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

	conversationv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/conversation/v1"
	messagev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/message/v1"
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
	reg := websocket.NewRegistry()
	memCache := membership.New()

	convClient, convConn := initConversationClient(cfg.ConversationSvcAddr, log)
	defer convConn.Close()

	msgClient, msgConn := initMessageClient(cfg.MessagingSvcAddr, log)
	defer msgConn.Close()

	presenceClient, presenceConn := initPresenceClient(cfg.PresenceSvcAddr, log)
	defer presenceConn.Close()

	disp := dispatcher.New(reg, memCache, presenceClient, rtr, instanceID, convClient)
	pw := presencewatcher.NewWatcher(redisClient, reg, memCache)
	pw.Start(ctx)

	rtr.Subscribe(ctx, disp.DeliverRemote)
	wsHandler := websocket.NewHandler(reg, presenceClient, convClient, msgClient, instanceID)

	// Kafka Consumer
	consumer := initKafka(ctx, cfg, disp, log)
	defer consumer.Close()

	// Servers
	obsSrv := initObservabilityServer(cfg, log)
	wsSrv := server.New(cfg.ReqHTTPAddr, initMainRouter(wsHandler))

	startServers(cfg, obsSrv, wsSrv, log)

	<-ctx.Done()
	performGracefulShutdown(obsSrv, wsSrv, reg, log)
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

func initConversationClient(addr string, log *zap.Logger) (conversationv1.ConversationApiClient, *grpc.ClientConn) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("failed to connect to conversation service", zap.Error(err))
	}
	return conversationv1.NewConversationApiClient(conn), conn
}

func initMessageClient(addr string, log *zap.Logger) (messagev1.MessageApiClient, *grpc.ClientConn) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("failed to connect to message service", zap.Error(err))
	}
	return messagev1.NewMessageApiClient(conn), conn
}

func initPresenceClient(addr string, log *zap.Logger) (presencev1.PresenceApiClient, *grpc.ClientConn) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("failed to connect to presence service", zap.Error(err))
	}
	return presencev1.NewPresenceApiClient(conn), conn
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
	return &http.Server{Addr: cfg.ObsHTTPAddr, Handler: mux}
}

func initMainRouter(wsHandler *websocket.Handler) http.Handler {
	mux := chi.NewRouter()
	mux.Handle("/ws", wsHandler)
	return mux
}

func startServers(cfg *config.Config, obsSrv *http.Server, wsSrv *server.Server, log *zap.Logger) {
	go func() {
		log.Info("starting observability server", zap.String("addr", cfg.ObsHTTPAddr))
		if err := obsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("observability server error", zap.Error(err))
		}
	}()
	go func() {
		log.Info("starting main server", zap.String("addr", cfg.ReqHTTPAddr))
		if err := wsSrv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()
}

func performGracefulShutdown(obs *http.Server, ws *server.Server, reg *websocket.Registry, log *zap.Logger) {
	log.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ws.Shutdown(ctx); err != nil {
		log.Error("error during main server shutdown", zap.Error(err))
	}
	if err := obs.Shutdown(ctx); err != nil {
		log.Error("error during observability server shutdown", zap.Error(err))
	}
	reg.CloseAll()
	log.Info("shutdown complete, exiting")
}
