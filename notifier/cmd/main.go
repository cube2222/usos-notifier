package main

import (
	"log"
	"net/http"

	"github.com/cube2222/grpc-utils/logger"
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
	m.Use(requestid.HTTPInterceptor)
	m.Use(logger.HTTPInjector(logger.NewStdLogger(), requestid.Key))
	m.Use(logger.HTTPLogger())
	m.HandleFunc("/notifier/webhook", s.HandleWebhookHTTP)
	log.Println("Serving...")
	log.Fatal(http.ListenAndServe(":8080", m))
}
