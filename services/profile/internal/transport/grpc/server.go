package grpc

import (
	"context"
	"net"

	profilev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/profile/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/observability"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/service"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	grpcServer *grpc.Server
	svc        *service.ProfileService
}

func NewServer(svc *service.ProfileService) *Server {
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	h := NewHandler(svc)
	profilev1.RegisterProfileApiServer(grpcServer, h)

	return &Server{
		grpcServer: grpcServer,
		svc:        svc,
	}
}

func (s *Server) Start(port string) error {
	lisAddr := port

	lis, err := net.Listen("tcp", lisAddr)
	if err != nil {
		return err
	}

	observability.GetLogger(context.Background()).Info("profile gRPC server listening", zap.String("port", port))
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}
