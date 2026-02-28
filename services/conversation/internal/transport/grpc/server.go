package grpc

import (
	"log"
	"net"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	conversationv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/conversation/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/application"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/auth"
)

type Server struct {
	conversationv1.UnimplementedConversationApiServer
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

	conversationv1.RegisterConversationApiServer(
		grpcServer,
		s,
	)

	return s
}

func (s *Server) Start(port string) {
	// If port starts with ":", use it directly. If not, prepend ":"
	lisAddr := port

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
