package grpc

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	messagingv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/messaging/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/application"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/auth"
)

type Server struct {
	messagingv1.UnimplementedMessagingApiServer
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

	messagingv1.RegisterMessagingApiServer(
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

	go func() {
		log.Println("gRPC listening on", port)
		if err := s.grpcServer.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("shutting down gRPC...")
	s.grpcServer.GracefulStop()
}
