package notifier

import (
	"context"
	"encoding/json"

	"github.com/cube2222/usos-notifier/common/events/publisher"
	"github.com/cube2222/usos-notifier/common/users"

	"github.com/pkg/errors"
)

type SendNotificationEvent struct {
	UserID  users.UserID `json:"user_id"`
	Message string       `json:"message"`
}

type NotificationSender interface {
	SendNotification(ctx context.Context, userID users.UserID, message string) error
}

type notificationSender struct {
	notificationsTopic string
	publisher          *publisher.Publisher
}

func NewNotificationSender(publisher *publisher.Publisher, notificationsTopic string) NotificationSender {
	return &notificationSender{
		notificationsTopic: notificationsTopic,
		publisher:          publisher,
	}
}

func (ns *notificationSender) SendNotification(ctx context.Context, userID users.UserID, message string) error {
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
