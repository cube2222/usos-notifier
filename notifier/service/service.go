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

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"github.com/cube2222/usos-notifier/common/events"
	"github.com/cube2222/usos-notifier/notifier"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"google.golang.org/api/option"
)

type Service struct {
	ds     *datastore.Client
	pubsub *pubsub.Client
	publisher *events.Publisher
	cli    *http.Client
}

func NewService() (*Service, error) {
	ds, err := datastore.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create datastore client")
	}

	cli, err := pubsub.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create pubsub client")
	}

	service := &Service{
		ds:     ds,
		pubsub: cli,
		publisher: events.NewPublisher(cli),
		cli:    http.DefaultClient,
	}

	go func() {
		log.Fatal(cli.Subscription("sender").Receive(context.Background(), service.handleMessageSendEvent))
	}()

	return service, nil
}

type MessageEvent struct {
	Sender struct {
		ID string `json:"id"`
	} `json:"sender"`
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient"`
	Timestamp int64 `json:"timestamp"`
	Message struct {
		Mid  string `json:"mid"`
		Text string `json:"text"`
		QuickReply struct {
			Payload string `json:"payload"`
		} `json:"quick_reply"`
	} `json:"message"`
}

type Webhook struct {
	Object string `json:"object"`
	Entry []struct {
		ID        string         `json:"id"`
		Time      int64          `json:"time"`
		Messaging []MessageEvent `json:"messaging"`
	} `json:"entry"`
}

func (s *Service) handleWebhook(ctx context.Context, webhook MessageEvent) error {
	userExists := true

	userID, err := s.getUserID(ctx, webhook.Sender.ID)
	if err != nil {
		if errors.Cause(err) == datastore.ErrNoSuchEntity {
			userExists = false
		} else {
			return errors.Wrap(err, "couldn't get userID")
		}
	}

	if !userExists {
		userID, err = s.createUser(ctx, webhook.Sender.ID)
		if err != nil {
			return errors.Wrap(err, "couldn't create user")
		}
	}

	err = s.publisher.PublishEvent(ctx, "commands",
		map[string]string{
			"user_id": userID,
			"origin":  "fb_messenger",
		},
		webhook.Message.Text,
	)
	if err != nil {
		return errors.Wrap(err, "couldn't publish event")
	}

	return nil
}

func (s *Service) HandleWebhookHTTP() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("hub.mode") == "subscribe" && r.URL.Query().Get("hub.verify_token") == "aowicb038qfi87uvabo8li7b32pv84743qv2" {
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
					// TODO: Proper zap logging
					log.Println(err)
					return
				}
			}
		}
	}
}

type UserID struct {
	UserID string `json:"user_id"`
}

type MessengerID struct {
	MessengerID string `json:"messenger_id"`
}

func (s *Service) createUser(ctx context.Context, messengerID string) (string, error) {
	userID, err := uuid.NewV4()
	if err != nil {
		return "", errors.Wrap(err, "couldn't generate uuid")
	}

	tx, err := s.ds.NewTransaction(ctx)
	if err != nil {
		return "", errors.Wrap(err, "couldn't begin transaction")
	}
	defer tx.Rollback()

	key1 := datastore.NameKey("mapping-userid-to-messenger", userID.String(), nil)
	key2 := datastore.NameKey("mapping-messenger-to-userid", messengerID, nil)

	_, err = tx.Put(key1, &MessengerID{messengerID})
	if err != nil {
		return "", errors.Wrap(err, "couldn't create userid to messenger mapping")
	}
	_, err = tx.Put(key2, &UserID{userID.String()})
	if err != nil {
		return "", errors.Wrap(err, "couldn't create messenger to userid mapping")
	}

	_, err = tx.Commit()
	if err != nil {
		return "", errors.Wrap(err, "couldn't commit transaction")
	}

	err = s.publisher.PublishEvent(ctx, "user-created",
		map[string]string{
			"origin": "fb-messenger",
		},
		userID.String(),
	)
	if err != nil {
		return "", errors.Wrap(err, "couldn't publish user created event")
		// TODO: Delete user
	}

	return userID.String(), nil
}

func (s *Service) getMessengerID(ctx context.Context, userID string) (string, error) {
	key := datastore.NameKey("mapping-userid-to-messenger", userID, nil)

	var out MessengerID
	err := s.ds.Get(ctx, key, &out)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get messengerID")
	}

	return out.MessengerID, nil
}

func (s *Service) getUserID(ctx context.Context, messengerID string) (string, error) {
	key := datastore.NameKey("mapping-messenger-to-userid", messengerID, nil)

	var out UserID
	err := s.ds.Get(ctx, key, &out)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get userID")
	}

	return out.UserID, nil
}

func (s *Service) handleMessageSendEvent(ctx context.Context, message *pubsub.Message) {
	defer message.Nack()

	event := notifier.SendNotificationEvent{}

	err := events.DecodeJSONMessage(message, &event)
	if err != nil {
		log.Println("couldn't pubsub message: ", err)
		return
	}

	messengerID, err := s.getMessengerID(ctx, event.UserID)
	if err != nil {
		log.Println("couldn't get messenger ID: ", err)
		return
	}

	err = s.sendMessage(ctx, messengerID, event.Message)
	if err != nil {
		log.Println("couldn't send message: ", err)
		return
	}

	message.Ack()
}

func (s *Service) sendMessage(ctx context.Context, messengerID string, body string) error {
	message := struct {
		MessagingType string `json:"messaging_type"`
		Recipient struct {
			ID string `json:"id"`
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
