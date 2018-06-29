package notifier

import (
	"context"

	"cloud.google.com/go/pubsub"
)

type SendNotificationEvent struct {
	UserID string `json:"user_id"`
	Message string `json:"message"`
}

type NotificationSender struct {
	pubsub *pubsub.Client
}

func NewNotificationSender() {

}

func (ns *NotificationSender) SendNotification(ctx context.Context, userID string, message string) {

}
