package subscriber

import (
	"context"

	"cloud.google.com/go/pubsub"
	"github.com/cube2222/grpc-utils/logger"
	"github.com/cube2222/usos-notifier/common/customerrors"
)

type Message struct {
	ID         string
	Data       []byte
	Attributes map[string]string
}

type HandlerFunc func(context.Context, *Message) error
type HandlerMiddleware func(f HandlerFunc) HandlerFunc

func Chain(f HandlerFunc, middleware ...HandlerMiddleware) HandlerFunc {
	handler := f
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}

func NewSubscriptionClient(client *pubsub.Client) *SubscriptionClient {
	return &SubscriptionClient{
		cli: client,
	}
}

type SubscriptionClient struct {
	cli *pubsub.Client
}

func (cli *SubscriptionClient) Subscribe(ctx context.Context, eventType string, handler HandlerFunc) error {
	return cli.cli.Subscription(eventType).Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		err := handler(ctx, &Message{
			ID:         msg.ID,
			Data:       msg.Data,
			Attributes: msg.Attributes,
		})
		if err != nil {
			if customerrors.IsPermanent(err) {
				msg.Ack()
				logger.FromContext(ctx).Errorf("Permanent error: %v", err)
			}
			msg.Nack()
		}
		msg.Ack()
	})
}
