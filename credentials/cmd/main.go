package main

import (
	"log"
	"net"
	"net/http"

	"github.com/cube2222/grpc-utils/health"
	"github.com/cube2222/grpc-utils/logger"
	"github.com/cube2222/grpc-utils/requestid"

	"github.com/cube2222/usos-notifier/credentials"
	"github.com/cube2222/usos-notifier/credentials/service"

	"github.com/go-chi/chi"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
)

func main() {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				requestid.ServerInterceptor(),
				logger.GRPCInjector(logger.NewStdLogger(), requestid.Key),
				logger.GRPCServerLogger(),
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
	m.Use(requestid.HTTPInterceptor)
	m.Use(logger.HTTPInjector(logger.NewStdLogger(), requestid.Key))
	m.Use(logger.HTTPLogger())
	m.HandleFunc("/credentials/authorization", s.ServeAuthorizationPageHTTP)
	m.HandleFunc("/credentials/authorize", s.HandleAuthorizeHTTP)

	go func() {
		log.Println("Serving...")
		log.Fatal(http.ListenAndServe(":8080", m))
	}()

	health.LaunchHealthCheckHandler()
}
