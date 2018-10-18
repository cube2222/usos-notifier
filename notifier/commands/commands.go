package commands

import (
	"context"
	"fmt"

	"github.com/cube2222/grpc-utils/logger"
	"github.com/cube2222/grpc-utils/requestid"
	"github.com/cube2222/usos-notifier/notifier"
	"github.com/pkg/errors"

	"github.com/cube2222/usos-notifier/common/events/subscriber"
	"github.com/cube2222/usos-notifier/common/users"
)

type CommandsHandler interface {
	Handle(matcher Matcher, handler func(ctx context.Context, userID users.UserID, params map[string]string) (string, error))
	HandleMessage(context.Context, *subscriber.Message) error
}

type HandleFunc func(ctx context.Context, userID users.UserID, params map[string]string) (string, error)

type commandsHandler struct {
	router *router
	sender notifier.NotificationSender
}

func NewCommandsHandler(sender notifier.NotificationSender) CommandsHandler {
	return &commandsHandler{
		router: &router{},
		sender: sender,
	}
}

type router struct {
	matchers []Matcher
	handlers []HandleFunc
}

func (r *router) addRoute(matcher Matcher, handler HandleFunc) {
	r.matchers = append(r.matchers, matcher)
	r.handlers = append(r.handlers, handler)
}

func (r *router) getHandler(text string) (handler HandleFunc, params map[string]string, err error) {
	for i, matcher := range r.matchers {
		params, err := matcher.Match(text)
		if err == ErrNoMatch {
			continue
		} else if err != nil {
			return nil, nil, errors.Wrap(err, "couldn't try to getHandler")
		}
		return r.handlers[i], params, nil
	}

	return nil, nil, ErrNoMatch
}

func (ch *commandsHandler) Handle(matcher Matcher, handler func(ctx context.Context, userID users.UserID, params map[string]string) (string, error)) {
	ch.router.addRoute(matcher, handler)
}

func (ch *commandsHandler) HandleMessage(ctx context.Context, msg *subscriber.Message) error {
	log := logger.FromContext(ctx)
	userID := users.NewUserID(msg.Attributes["user_id"])

	data, err := subscriber.DecodeTextMessage(msg)
	if err != nil {
		return subscriber.NewNonRetryableError(errors.Wrap(err, "couldn't decode text message"))
	}

	handler, params, err := ch.router.getHandler(string(data))
	if err != nil {
		log.Println("Omitting message. No match.")
		return nil
	}

	response, err := handler(ctx, userID, params)
	if err != nil {
		// TODO: Could add a few retries, and only notify about failure the last time
		publishErr := ch.sender.SendNotification(ctx, userID,
			fmt.Sprintf(
				"Przy obsłudze Twojej wiadomości coś poszło nie tak. Spróbuj jeszcze raz, albo skontaktuj się z nami, podając nam identyfikator wiadomości: %v",
				ctx.Value(requestid.Key),
			),
		)
		if publishErr != nil {
			return subscriber.NewNonRetryableError(errors.Wrap(publishErr, "error sending error notification"))
		}
		return subscriber.NewNonRetryableError(errors.Wrap(err, "error handling user message"))
	}

	if len(response) > 0 {
		err := ch.sender.SendNotification(ctx, userID, response)
		if err != nil {
			return errors.Wrap(err, "error sending response message")
		}
	}

	return nil
}
