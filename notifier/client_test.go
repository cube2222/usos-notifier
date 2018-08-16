package notifier

import (
	"context"
	"testing"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
)

func TestNotificationSender_SendNotification(t *testing.T) {
	cli, err := pubsub.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile("C:/Development/Projects/Go/src/github.com/cube2222/usos-notifier/usos-notifier-9a2e44d7f26b.json"))
	if err != nil {
		t.Fatal(err)
	}

	sender := NewNotificationSender(cli)

	err = sender.SendNotification(context.Background(), "me", "New method!")
	if err != nil {
		t.Fatal(err)
	}
}
