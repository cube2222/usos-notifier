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

type NotificationSender struct {
	publisher *publisher.Publisher
}

func NewNotificationSender(publisher *publisher.Publisher) *NotificationSender {
	return &NotificationSender{
		publisher: publisher,
	}
}

func (ns *NotificationSender) SendNotification(ctx context.Context, userID users.UserID, message string) error {
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
