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

type MessageReceived struct {
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

func (s *Service) handleWebhook(webhook MessageReceived) error {
	log.Println(fmt.Sprintf("%s: %s", webhook.Sender, webhook.Message.Text))
	return nil
}

func (s *Service) HandleWebhookHTTP() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		webhook := MessageReceived{}

		err := json.NewDecoder(r.Body).Decode(&webhook)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, err)
			return
		}

		err = s.handleWebhook(webhook)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			// TODO: Proper zap logging
			log.Println(err)
			return
		}
	}
}
