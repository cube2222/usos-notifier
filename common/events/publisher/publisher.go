package publisher

import (
	"context"
	"encoding/base64"

	"cloud.google.com/go/pubsub"
	"github.com/pkg/errors"
)

type PublishEventFunc func(ctx context.Context, eventType string, metadata map[string]string, message string) error
type PublishMiddleware func(f PublishEventFunc) PublishEventFunc

type Publisher struct {
	pubsub     *pubsub.Client
	topics     map[string]*pubsub.Topic
	middleware []PublishMiddleware
}

func NewPublisher(cli *pubsub.Client) *Publisher {
	return &Publisher{
		pubsub:     cli,
		topics:     make(map[string]*pubsub.Topic),
		middleware: []PublishMiddleware{},
	}
}

func (p *Publisher) PublishEvent(ctx context.Context, eventType string, metadata map[string]string, message string) error {
	publisher := p.publishEvent

	for i := len(p.middleware) - 1; i >= 0; i-- {
		publisher = p.middleware[i](publisher)
	}

	return publisher(ctx, eventType, metadata, message)
}

func (p *Publisher) publishEvent(ctx context.Context, eventType string, metadata map[string]string, message string) error {
	res := p.getTopic(eventType).Publish(ctx, &pubsub.Message{
		Data:       []byte(base64.StdEncoding.EncodeToString([]byte(message))),
		Attributes: metadata,
	})

	_, err := res.Get(ctx)

	return errors.Wrapf(err, "couldn't publish event of type %v", eventType)
}

func (p *Publisher) getTopic(topic string) *pubsub.Topic {
	t, ok := p.topics[topic]
	if !ok {
		t = p.pubsub.Topic(topic)
		p.topics[topic] = t
	}

	return t
}

func (p *Publisher) Use(f PublishMiddleware) *Publisher {
	p.middleware = append(p.middleware, f)
	return p
}
