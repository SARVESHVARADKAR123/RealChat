package grpc

import (
	"log"
	"net"

	authv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/auth/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/service"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

type Server struct {
	grpcServer *grpc.Server
	svc        *service.AuthService
}

func NewServer(svc *service.AuthService) *Server {
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	h := NewHandler(svc)
	authv1.RegisterAuthApiServer(grpcServer, h)

	return &Server{
		grpcServer: grpcServer,
		svc:        svc,
	}
}

func (s *Server) Start(port string) error {
	lisAddr := port
	if len(port) > 0 && port[0] != ':' {
		lisAddr = ":" + port
	}

	lis, err := net.Listen("tcp", lisAddr)
	if err != nil {
		return err
	}

	log.Printf("auth gRPC server listening on %s", port)
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}
