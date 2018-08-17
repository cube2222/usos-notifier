package main

import (
	"context"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"

	gdatastore "cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"github.com/cube2222/usos-notifier/common/events/publisher"
	"github.com/cube2222/usos-notifier/common/events/subscriber"
	"github.com/cube2222/usos-notifier/credentials/resources"
	"github.com/cube2222/usos-notifier/credentials/service/datastore"
	"github.com/cube2222/usos-notifier/notifier"
	"github.com/go-chi/chi"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudkms/v1"
	"google.golang.org/api/option"
	"google.golang.org/grpc"

	"github.com/cube2222/grpc-utils/health"
	"github.com/cube2222/grpc-utils/logger"
	"github.com/cube2222/grpc-utils/requestid"

	"github.com/cube2222/usos-notifier/credentials"
	"github.com/cube2222/usos-notifier/credentials/service"
)

func main() {
	httpCli, err := google.DefaultClient(context.Background(), cloudkms.CloudPlatformScope)
	if err != nil {
		log.Fatal("Couldn't setup google default http client: ", err)
	}
	kms, err := cloudkms.New(httpCli)
	if err != nil {
		log.Fatal("Couldn't create cloud kms client", err)

	}
	ds, err := gdatastore.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		log.Fatal("Couldn't create datastore client", err)

	}
	pubsubCli, err := pubsub.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		log.Fatal("Couldn't create datastore client", err)

	}

	credentialsStorage := datastore.NewCredentialsStorage(ds, kms)
	tokenStorage := datastore.NewTokenStorage(ds)
	notificationSender := notifier.NewNotificationSender(
		publisher.
			NewPublisher(pubsubCli).
			Use(publisher.WithRequestID),
	)

	data, err := resources.Asset("authorize.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl, err := template.New("authorize.html").Parse(string(data))
	if err != nil {
		log.Fatal(err)
	}

	s, err := service.NewService(credentialsStorage, tokenStorage, notificationSender, tmpl)
	if err != nil {
		log.Fatal(err, "Couldn't create service")
	}

	// Set up grpc usos sessions service
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				requestid.ServerInterceptor(),
				logger.GRPCInjector(logger.NewStdLogger(), requestid.Key),
				logger.GRPCServerLogger(),
			),
		),
	)
	credentials.RegisterCredentialsServer(server, s)
	lis, err := net.Listen("tcp", ":8081")
	go func() {
		log.Fatal(server.Serve(lis))
	}()

	// Set up authorization page handler
	m := chi.NewMux()
	m.Use(requestid.HTTPInterceptor)
	m.Use(logger.HTTPInjector(logger.NewStdLogger(), requestid.Key))
	m.Use(logger.HTTPLogger())
	m.HandleFunc("/credentials/authorization", s.HandleAuthorizationPageHTTP)
	m.HandleFunc("/credentials/authorize", s.HandleAuthorizeHTTP)
	go func() {
		log.Println("Serving...")
		log.Fatal(http.ListenAndServe(":8080", m))
	}()

	// Set up user created event subscription
	go func() {
		log.Fatal(
			subscriber.
				NewSubscriptionClient(pubsubCli).
				Subscribe(
					context.Background(),
					"credentials-notifier-user_created",
					subscriber.Chain(
						s.HandleUserCreatedEvent,
						subscriber.WithLogger(logger.NewStdLogger()),
						subscriber.WithRequestID,
						subscriber.WithLogging(requestid.Key),
					),
				),
		)
	}()

	// Set up health checking
	health.LaunchHealthCheckHandler()
}
