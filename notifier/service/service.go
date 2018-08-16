// TODO: Create sub-packages for the mapping and messaging

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/cube2222/grpc-utils/logger"
	"github.com/cube2222/grpc-utils/requestid"
	"github.com/cube2222/usos-notifier/common/customerrors"

	"github.com/cube2222/usos-notifier/common/events/publisher"
	"github.com/cube2222/usos-notifier/common/events/subscriber"
	"github.com/cube2222/usos-notifier/notifier"
	"github.com/cube2222/usos-notifier/notifier/service/datastore"

	gdatastore "cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"github.com/pkg/errors"
	"google.golang.org/api/option"
)

type Service struct {
	userMapping notifier.UserMapping
	pubsub      *pubsub.Client
	publisher   *publisher.Publisher
	cli         *http.Client
}

func NewService() (*Service, error) {
	ds, err := gdatastore.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create datastore client")
	}

	cli, err := pubsub.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create pubsub client")
	}

	service := &Service{
		userMapping: datastore.NewUserMapping(ds),
		pubsub:      cli,
		publisher: publisher.
			NewPublisher(cli).
			Use(publisher.WithRequestID),
		cli: http.DefaultClient,
	}

	go func() {
		log.Fatal(
			subscriber.
				NewSubscriptionClient(cli).
				Subscribe(
					context.Background(),
					"notifier-notifications",
					subscriber.Chain(
						service.handleMessageSendEvent,
						subscriber.WithLogger(logger.NewStdLogger()),
						subscriber.WithRequestID,
						subscriber.WithLogging(requestid.Key),
					),
				),
		)
	}()

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

func (s *Service) HandleWebhookHTTP(w http.ResponseWriter, r *http.Request) {
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
			err = s.handleWebhook(r.Context(), event)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println(err)
				return
			}
		}
	}
}

func (s *Service) handleWebhook(ctx context.Context, webhook MessageEvent) error {
	userExists := true

	userID, err := s.userMapping.GetUserID(ctx, webhook.Sender.ID)
	if err != nil {
		if customerrors.IsPermanent(err) {
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

func (s *Service) handleMessageSendEvent(ctx context.Context, message *subscriber.Message) error {
	event := notifier.SendNotificationEvent{}

	err := subscriber.DecodeJSONMessage(message, &event)
	if err != nil {
		return errors.Wrap(err, "couldn't decode json message")
	}

	messengerID, err := s.userMapping.GetMessengerID(ctx, event.UserID)
	if err != nil {
		return errors.Wrap(err, "couldn't get messenger ID")
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
