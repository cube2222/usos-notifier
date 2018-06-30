package events

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"cloud.google.com/go/pubsub"
	"github.com/pkg/errors"
)

type Publisher struct {
	pubsub *pubsub.Client
	topics map[string]*pubsub.Topic
}

func NewPublisher(cli *pubsub.Client) *Publisher {
	return &Publisher{
		pubsub: cli,
		topics: make(map[string]*pubsub.Topic),
	}
}

func (p *Publisher) PublishEvent(ctx context.Context, eventType string, metadata map[string]string, message string) error {
	// TODO: Create context to attributes function to pass request_id
	// TODO: Create one topic instance
	res := p.getTopic(eventType).Publish(ctx, &pubsub.Message{
		Data:       []byte(base64.StdEncoding.EncodeToString([]byte(message))),
		Attributes: metadata,
	})

	_, err := res.Get(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't publish message")
	}

	return nil
}

func (p *Publisher) getTopic(topic string) *pubsub.Topic {
	t, ok := p.topics[topic]
	if !ok {
		t = p.pubsub.Topic(topic)
		p.topics[topic] = t
	}

	return t
}

func DecodeJSONMessage(message *pubsub.Message, dst interface{}) error {
	data, err := DecodeTextMessage(message)
	if err != nil {
		return errors.Wrap(err, "couldn't base64 decode message")
	}

	err = json.Unmarshal(data, dst)
	if err != nil {
		return errors.Wrap(err, "couldn't unmarshal send message event")
	}

	return nil
}

func DecodeTextMessage(message *pubsub.Message) ([]byte, error) {
	return base64.StdEncoding.DecodeString(string(message.Data))
}
