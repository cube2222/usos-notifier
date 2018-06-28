package main

import (
	"log"
	"net/http"

	"github.com/cube2222/usos-notifier/notifier/service"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/acme/autocert"
)

func main() {
	s, err := service.NewService()
	if err != nil {
		log.Fatal(err)
	}

	m := mux.NewRouter()
	m.HandleFunc("/webhook", s.HandleWebhookHTTP())
	http.ListenAndServe(":80", m)
	log.Fatal(http.Serve(autocert.NewListener("notifier.jacobmartins.com"), m))
}
