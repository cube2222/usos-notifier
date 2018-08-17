package main

import (
	"context"
	"log"
	"net/http"
	"os"

	gdatastore "cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"github.com/cube2222/usos-notifier/common/events/subscriber"
	"github.com/go-chi/chi"
	"google.golang.org/api/option"

	"github.com/cube2222/grpc-utils/logger"
	"github.com/cube2222/grpc-utils/requestid"

	"github.com/cube2222/usos-notifier/common/events/publisher"
	"github.com/cube2222/usos-notifier/notifier/service"
	"github.com/cube2222/usos-notifier/notifier/service/datastore"
)

// TODO: Add config.
func main() {
	ds, err := gdatastore.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		log.Fatal("Couldn't create datastore client: ", err)
	}
	pubsubCli, err := pubsub.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		log.Fatal("Couldn't create pubsub client: ", err)
	}

	s, err := service.NewService(
		datastore.NewUserMapping(ds),
		publisher.
			NewPublisher(pubsubCli).
			Use(publisher.WithRequestID),
		service.NewMessengerRateLimiter(100, 1000),
	)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Fatal(
			subscriber.
				NewSubscriptionClient(pubsubCli).
				Subscribe(
					context.Background(),
					"notifier-notifications",
					subscriber.Chain(
						s.HandleMessageSendEvent,
						subscriber.WithLogger(logger.NewStdLogger()),
						subscriber.WithRequestID,
						subscriber.WithLogging(requestid.Key),
					),
				),
		)
	}()

	m := chi.NewMux()
	m.Use(requestid.HTTPInterceptor)
	m.Use(logger.HTTPInjector(logger.NewStdLogger(), requestid.Key))
	m.Use(logger.HTTPLogger())
	m.HandleFunc("/notifier/webhook", s.HandleMessageReceivedWebhookHTTP)
	log.Println("Serving...")
	log.Fatal(http.ListenAndServe(":8080", m))
}
