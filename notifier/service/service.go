// TODO: Create sub-packages for the mapping and messaging

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/cube2222/grpc-utils/logger"

	"github.com/cube2222/usos-notifier/common/events/publisher"
	"github.com/cube2222/usos-notifier/common/events/subscriber"
	"github.com/cube2222/usos-notifier/notifier"
)

type Service struct {
	userMapping          notifier.UserMapping
	publisher            *publisher.Publisher
	cli                  *http.Client
	messengerRateLimiter *MessengerRateLimiter
}

func NewService(mapping notifier.UserMapping, publisher *publisher.Publisher, limiter *MessengerRateLimiter) (*Service, error) {
	service := &Service{
		userMapping:          mapping,
		publisher:            publisher,
		cli:                  http.DefaultClient,
		messengerRateLimiter: limiter,
	}

	return service, nil
}

type Webhook struct {
	Object string `json:"object"`
	Entry  []struct {
		ID        string         `json:"id"`
		Time      int64          `json:"time"`
		Messaging []MessageEvent `json:"messaging"`
	} `json:"entry"`
}

type MessageEvent struct {
	Sender struct {
		ID notifier.MessengerID `json:"id"`
	} `json:"sender"`
	Recipient struct {
		ID notifier.MessengerID `json:"id"`
	} `json:"recipient"`
	Timestamp int64 `json:"timestamp"`
	Message   struct {
		Mid        string `json:"mid"`
		Text       string `json:"text"`
		QuickReply struct {
			Payload string `json:"payload"`
		} `json:"quick_reply"`
	} `json:"message"`
}

func (s *Service) HandleMessageReceivedWebhookHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	//TODO Change verify token to secret-based
	if r.URL.Query().Get("hub.mode") == "subscribe" && r.URL.Query().Get("hub.verify_token") == "aowicb038qfi87uvabo8li7b32pv84743qv2" {
		log.Printf("Handling challange.")
		fmt.Fprint(w, r.URL.Query().Get("hub.challenge"))
		return
	}

	webhook := Webhook{}

	err := json.NewDecoder(r.Body).Decode(&webhook)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)
		return
	}

	for _, page := range webhook.Entry {
		for _, event := range page.Messaging {
			err = s.handleMessageReceived(r.Context(), event)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println(err)
				return
			}
		}
	}
}

func (s *Service) handleMessageReceived(ctx context.Context, webhook MessageEvent) error {
	log := logger.FromContext(ctx)

	rateLimit, limited := s.messengerRateLimiter.LimitMessengerUser(webhook.Sender.ID)
	if limited {
		switch rateLimit.Reason {
		case ReasonUser:
			err := s.sendMessage(ctx, webhook.Sender.ID, fmt.Sprintf("Dostałem od Ciebie za dużo wiadomości. Spróbuj ponownie za %d minut.", int(rateLimit.TimeLeft.Round(time.Minute).Minutes())))
			if err != nil {
				log.Printf("Couldn't send rate limit notification: %v", err)
			}
		case ReasonGeneral:
			err := s.sendMessage(ctx, webhook.Sender.ID, fmt.Sprintf("Jestem akurat przytłoczony ilością wiadomości od użytkowników. Spróbuj ponownie za %d minut.", int(rateLimit.TimeLeft.Round(time.Minute).Minutes())))
			if err != nil {
				log.Printf("Couldn't send rate limit notification: %v", err)
			}
		default:
			err := s.sendMessage(ctx, webhook.Sender.ID, fmt.Sprintf("Nie mogę w tym momencie obsłużyć Twojej wiadomości. Spróbuj ponownie za %d minut.", int(rateLimit.TimeLeft.Round(time.Minute).Minutes())))
			if err != nil {
				log.Printf("Couldn't send rate limit notification: %v", err)
			}
		}
		log.Printf("Rate limiting messenger user %v because of %v", webhook.Sender.ID, rateLimit.Reason)
		return nil
	}

	userExists := true

	userID, err := s.userMapping.GetUserID(ctx, webhook.Sender.ID)
	if err != nil {
		if err == notifier.ErrNotFound {
			userExists = false
		} else {
			return errors.Wrap(err, "couldn't get userID")
		}
	}

	if !userExists {
		userID, err = s.userMapping.CreateUser(ctx, webhook.Sender.ID)
		if err != nil {
			return errors.Wrap(err, "couldn't create user")
		}

		err = s.publisher.PublishEvent(ctx, "notifier-user_created",
			map[string]string{
				"origin": "fb_messenger",
			},
			userID.String(),
		)
		if err != nil {
			return errors.Wrap(err, "couldn't publish user created event")
		}

	}

	err = s.publisher.PublishEvent(ctx, "notifier-commands",
		map[string]string{
			"user_id": userID.String(),
			"origin":  "fb_messenger",
		},
		webhook.Message.Text,
	)
	if err != nil {
		return errors.Wrap(err, "couldn't publish event")
	}

	return nil
}

func (s *Service) HandleMessageSendEvent(ctx context.Context, message *subscriber.Message) error {
	event := notifier.SendNotificationEvent{}

	err := subscriber.DecodeJSONMessage(message, &event)
	if err != nil {
		return subscriber.NewNonRetryableError(errors.Wrap(err, "couldn't decode json message"))
	}

	messengerID, err := s.userMapping.GetMessengerID(ctx, event.UserID)
	if err != nil {
		out := errors.Wrap(err, "couldn't get messenger ID")
		if err == notifier.ErrNotFound {
			return subscriber.NewNonRetryableError(out)
		}
		return out
	}

	err = s.sendMessage(ctx, messengerID, event.Message)
	if err != nil {
		return errors.Wrap(err, "couldn't send message")
	}

	return nil
}

func (s *Service) sendMessage(ctx context.Context, messengerID notifier.MessengerID, body string) error {
	message := struct {
		MessagingType string `json:"messaging_type"`
		Recipient     struct {
			ID notifier.MessengerID `json:"id"`
		} `json:"recipient"`
		Message struct {
			Text string `json:"text"`
		} `json:"message"`
	}{}
	message.MessagingType = "UPDATE"
	message.Recipient.ID = messengerID
	message.Message.Text = body

	fbURL, err := url.Parse("https://graph.facebook.com/v2.6/me/messages")
	if err != nil {
		return errors.Wrap(err, "couldn't parse fb url")
	}

	query := fbURL.Query()
	query.Set("access_token", os.Getenv("MESSENGER_API"))
	fbURL.RawQuery = query.Encode()

	data, err := json.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "couldn't encode message as json")
	}

	req, err := http.NewRequest("POST", fbURL.String(), bytes.NewReader(data))
	if err != nil {
		return errors.Wrap(err, "couldn't create new request")
	}

	req.Header.Add("content-type", "application/json")

	res, err := s.cli.Do(req)
	if err != nil {
		return errors.Wrap(err, "couldn't make http request")
	}

	if res.StatusCode != http.StatusOK {
		data, _ := ioutil.ReadAll(res.Body)
		return errors.Errorf("received status code %d when posting to fb messenger API: %s", res.StatusCode, data)
	}

	return nil
}
