package main

import (
	"log"
	"net"

	"github.com/cube2222/grpc-utils/health"
	"github.com/cube2222/grpc-utils/requestid"
	"github.com/cube2222/usos-notifier/credentials"
	"github.com/cube2222/usos-notifier/credentials/service"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Couldn't set up logger: %v", err)
	}
	defer logger.Sync()

	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				grpc_ctxtags.UnaryServerInterceptor(),
				requestid.ServerInterceptor(),
				grpczap.UnaryServerInterceptor(logger),
			),
		),
	)

	s, err := service.NewService()
	if err != nil {
		log.Fatal(err, "Couldn't create service")
	}

	credentials.RegisterCredentialsServer(server, s)

	lis, err := net.Listen("tcp", ":8080")

	log.Println("Serving...")
	go log.Fatal(server.Serve(lis))
	health.LaunchHealthCheckHandler()
}
