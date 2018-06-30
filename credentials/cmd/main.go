package main

import (
	"log"
	"net"
	"net/http"

	"github.com/cube2222/grpc-utils/health"
	"github.com/cube2222/usos-notifier/credentials"
	"github.com/cube2222/usos-notifier/credentials/service"
	"github.com/go-chi/chi"
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
				grpczap.UnaryServerInterceptor(logger),
			),
		),
	)

	s, err := service.NewService()
	if err != nil {
		log.Fatal(err, "Couldn't create service")
	}

	credentials.RegisterCredentialsServer(server, s)

	lis, err := net.Listen("tcp", ":8081")

	go func() {
		log.Fatal(server.Serve(lis))
	}()

	m := chi.NewMux()
	m.HandleFunc("/credentials/authorization", s.ServeAuthorizationPageHTTP)
	m.HandleFunc("/credentials/authorize", s.HandleAuthorizeHTTP)

	go func() {
		log.Fatal(http.ListenAndServe(":8080", m))
	}() // TODO: TLS

	health.LaunchHealthCheckHandler()
}
