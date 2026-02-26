package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/observability"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/clients"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/config"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/handlers"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/router"
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

	authConn := mustDial(cfg.AuthGRPCAddr)
	profileConn := mustDial(cfg.ProfileGRPCAddr)
	msgConn := mustDial(cfg.MessagingGRPCAddr)
	convConn := mustDial(cfg.ConversationGRPCAddr)
	presenceConn := mustDial(cfg.PresenceGRPCAddr)

	defer authConn.Close()
	defer profileConn.Close()
	defer msgConn.Close()
	defer convConn.Close()
	defer presenceConn.Close()

	// HTTP Server for Observability (Metrics & Health)
	obsMux := http.NewServeMux()
	obsMux.Handle("/metrics", promhttp.Handler())
	obsMux.Handle("/health/live", http.HandlerFunc(observability.HealthLiveHandler))
	obsMux.Handle("/health/ready", observability.HealthReadyHandler())

	obsSrv := &http.Server{Addr: cfg.ObsHTTPAddr, Handler: obsMux}
	go func() {
		log.Info("HTTP observability server started", zap.String("addr", cfg.ObsHTTPAddr))
		if err := obsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP observability server failed", zap.Error(err))
		}
	}()

	factory := clients.NewFactory(authConn, profileConn, convConn, msgConn, presenceConn)

	authH := handlers.NewAuthHandler(factory.Auth)
	profileH := handlers.NewProfileHandler(factory.Profile)
	convH := handlers.NewConversationHandler(factory.Conversation, factory.Profile)

	msgH := handlers.NewMessageHandler(factory.Message)
	presenceH := handlers.NewPresenceHandler(factory.Presence)

	r := router.NewRouter(authH, profileH, convH, msgH, presenceH, cfg)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	go func() {
		log.Info("gateway started", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("gateway shutdown failed", zap.Error(err))
	}
	if err := obsSrv.Shutdown(ctx); err != nil {
		log.Error("observability shutdown failed", zap.Error(err))
	}

	log.Info("gateway stopped")
}

func mustDial(addr string) *grpc.ClientConn {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  200 * time.Millisecond,
				Multiplier: 1.6,
				Jitter:     0.2,
				MaxDelay:   3 * time.Second,
			},
			MinConnectTimeout: 3 * time.Second,
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                60 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)

	if err != nil {
		observability.Log.Fatal("dial failed", zap.String("addr", addr), zap.Error(err))
	}

	return conn
}
