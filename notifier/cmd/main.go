package main

import (
	"log"
	"net/http"

	"github.com/cube2222/grpc-utils/httplogger"
	"github.com/cube2222/grpc-utils/requestid"
	"github.com/cube2222/usos-notifier/notifier/service"
	"github.com/go-chi/chi"
)

func main() {
	s, err := service.NewService()
	if err != nil {
		log.Fatal(err)
	}

	m := chi.NewMux()
	m.Use(requestid.HTTPInjector)
	m.Use(httplogger.HTTPInject)
	m.HandleFunc("/webhook", s.HandleWebhookHTTP())
	log.Println("Serving...")
	log.Fatal(http.ListenAndServeTLS(":443", "cert.crt", "cert.key", m))
}
