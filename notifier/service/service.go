package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Service struct {
}

func NewService() (*Service, error) {

	return &Service{
	}, nil
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

func (s *Service) handleWebhook(webhook MessageEvent) error {
	log.Println(fmt.Sprintf("%s: %s", webhook.Sender, webhook.Message.Text))
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
				err = s.handleWebhook(event)
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
