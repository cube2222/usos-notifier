package main

import (
	"log"
	"net/http"

	"github.com/cube2222/usos-notifier/notifier/service"
	"github.com/gorilla/mux"
)

func main() {
	s, err := service.NewService()
	if err != nil {
		log.Fatal(err)
	}

	m := mux.NewRouter()
	m.HandleFunc("/webhook", s.HandleWebhookHTTP())
	http.ListenAndServeTLS(":443", "cert.crt", "cert.key", m)
}
