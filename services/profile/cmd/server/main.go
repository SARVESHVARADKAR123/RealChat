package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/cache"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/config"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/handler"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/kafka"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/observability"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/outbox"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/repository"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/service"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/transport/grpc"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
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
	obsMux := chi.NewRouter()
	obsMux.Use(observability.MetricsMiddleware(cfg.ServiceName))
	obsMux.Handle("/metrics", promhttp.Handler())
	obsMux.Get("/health/live", observability.HealthLiveHandler)
	// We'll set ready handler later after DB is init

	go func() {
		log.Info("HTTP observability server started", zap.String("addr", cfg.HTTPAddr))
		if err := http.ListenAndServe(cfg.HTTPAddr, obsMux); err != nil {
			log.Error("HTTP observability server failed", zap.Error(err))
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database
	db, err := repository.NewDB(ctx, cfg.DATABASE_URL)
	if err != nil {
		log.Fatal("db open failed", zap.Error(err))
	}

	// Update observability health with live DB
	obsMux.Get("/health/ready", observability.HealthReadyHandler(db))

	// Redis
	rdb := cache.New(cfg.RedisAddr)

	// Repositories
	profileRepo := &repository.ProfileRepo{DB: db}
	contactRepo := &repository.ContactRepo{DB: db}
	blockRepo := &repository.BlockRepo{DB: db}
	outboxRepo := outbox.NewRepository(db)

	// Services
	profileSvc := &service.ProfileService{
		Repo:   profileRepo,
		Cache:  &cache.ProfileCache{R: rdb},
		Outbox: outboxRepo,
	}
	contactSvc := &service.ContactService{
		Repo:      contactRepo,
		BlockRepo: blockRepo,
		Outbox:    outboxRepo,
	}
	blockSvc := &service.BlockService{
		Repo:        blockRepo,
		ContactRepo: contactRepo,
		Outbox:      outboxRepo,
	}

	// Kafka producer + outbox publisher
	producer := kafka.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	publisher := outbox.NewPublisher(outboxRepo, producer)
	go publisher.Start(ctx)

	// Kafka consumer — auto-create profile on user registration
	go kafka.StartUserCreatedConsumer(ctx, cfg.KafkaBrokers, profileRepo)

	// HTTP server
	mux := handler.NewRouter(cfg, profileSvc, contactSvc, blockSvc, db)
	srv := &http.Server{Addr: ":" + cfg.HTTPPort, Handler: mux}

	// gRPC server
	grpcSrv := grpc.NewServer(profileSvc)

	go func() {
		log.Info("profile HTTP started", zap.String("port", cfg.HTTPPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	go func() {
		if err := grpcSrv.Start(cfg.GRPC_ADDR); err != nil {
			log.Fatal("gRPC server error", zap.String("addr", cfg.GRPC_ADDR), zap.Error(err))
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Info("received signal, initiating shutdown")
	cancel() // stop outbox publisher + kafka consumer

	ctxShut, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()

	_ = srv.Shutdown(ctxShut)
	grpcSrv.Stop()
	log.Info("profile stopped")
}
