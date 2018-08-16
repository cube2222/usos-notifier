package notifier

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/pubsub"
	"github.com/cube2222/usos-notifier/common/events/publisher"
	"github.com/pkg/errors"
)

type SendNotificationEvent struct {
	UserID  UserID `json:"user_id"`
	Message string `json:"message"`
}

type NotificationSender struct {
	publisher *publisher.Publisher
}

func NewNotificationSender(client *pubsub.Client) *NotificationSender {
	return &NotificationSender{
		publisher: publisher.NewPublisher(client),
	}
}

func (ns *NotificationSender) SendNotification(ctx context.Context, userID UserID, message string) error {
	data, err := json.Marshal(SendNotificationEvent{
		UserID:  userID,
		Message: message,
	})
	if err != nil {
		return errors.Wrap(err, "couldn't marshal send notification event")
	}

	err = ns.publisher.PublishEvent(ctx, "notifications", nil, string(data))
	if err != nil {
		return errors.Wrap(err, "couldn't publish event")
	}

	return nil
}
