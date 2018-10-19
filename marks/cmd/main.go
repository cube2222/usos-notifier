package main

import (
	"context"
	"log"

	gdatastore "cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"github.com/cube2222/grpc-utils/health"
	"github.com/cube2222/grpc-utils/logger"
	"github.com/cube2222/grpc-utils/requestid"
	"github.com/cube2222/usos-notifier/common/events/publisher"
	"github.com/cube2222/usos-notifier/common/events/subscriber"
	"github.com/cube2222/usos-notifier/credentials"
	"github.com/cube2222/usos-notifier/marks"
	"github.com/cube2222/usos-notifier/marks/service"
	"github.com/cube2222/usos-notifier/marks/service/datastore"
	"github.com/cube2222/usos-notifier/notifier"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func main() {
	config := &marks.Config{}
	envconfig.MustProcess("marks", config)

	ds, err := gdatastore.NewClient(context.Background(), config.ProjectName, option.WithCredentialsFile(config.GoogleApplicationCredentials))
	if err != nil {
		log.Fatal("Couldn't create datastore client", err)
	}
	pubsubCli, err := pubsub.NewClient(context.Background(), config.ProjectName, option.WithCredentialsFile(config.GoogleApplicationCredentials))
	if err != nil {
		log.Fatal("Couldn't create pubsub client", err)
	}

	conn, err := grpc.Dial(config.CredentialsAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	credentialsCli := credentials.NewCredentialsClient(conn)
	userStorage := datastore.NewUserStorage(ds)
	pub := publisher.
		NewPublisher(pubsubCli).
		Use(publisher.WithRequestID)
	notificationSender := notifier.NewNotificationSender(
		pub,
		config.NotificationsTopic,
	)

	s := service.NewService(credentialsCli, notificationSender, userStorage)

	// Set up user message event subscription
	go func() {
		log.Fatal(
			subscriber.
				NewSubscriptionClient(pubsubCli).
				Subscribe(
					context.Background(),
					config.CommandsSubscription,
					subscriber.Chain(
						s.HandleUserMessageEvent,
						subscriber.WithLogger(logger.NewStdLogger()),
						subscriber.WithRequestID,
						subscriber.WithLogging(requestid.Key),
					),
				),
		)
	}()

	// Set up credentials received event subscription
	go func() {
		log.Fatal(
			subscriber.
				NewSubscriptionClient(pubsubCli).
				Subscribe(
					context.Background(),
					config.CredentialsReceivedSubscription,
					subscriber.Chain(
						s.HandleCredentialsProvidedEvent,
						subscriber.WithLogger(logger.NewStdLogger()),
						subscriber.WithRequestID,
						subscriber.WithLogging(requestid.Key),
					),
				),
		)
	}()

	go s.RunScoreChecker(context.Background())

	// Set up health checking
	health.LaunchHealthCheckHandler()
}
