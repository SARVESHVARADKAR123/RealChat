package grpc

import (
	"log"
	"net"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	messagev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/message/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/application"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/auth"
)

type Server struct {
	messagev1.UnimplementedMessageApiServer
	grpcServer *grpc.Server
	app        *application.Service
}

func New(app *application.Service) *Server {
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(auth.Interceptor),
	)

	s := &Server{
		grpcServer: grpcServer,
		app:        app,
	}

	messagev1.RegisterMessageApiServer(
		grpcServer,
		s,
	)

	return s
}

func (s *Server) Start(port string) {
	// If port starts with ":", use it directly. If not, prepend ":"
	lisAddr := port
	if len(port) > 0 && port[0] != ':' {
		lisAddr = ":" + port
	}

	lis, err := net.Listen("tcp", lisAddr)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("gRPC listening on", port)
	if err := s.grpcServer.Serve(lis); err != nil {
		log.Println("gRPC server stopped:", err)
	}
}

func (s *Server) Stop() {
	log.Println("shutting down gRPC...")
	s.grpcServer.GracefulStop()
}
