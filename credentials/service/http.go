package service

import (
	"fmt"
	"net/http"

	"github.com/cube2222/grpc-utils/logger"
)

func (s *Service) HandleAuthorizationPageHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	matched := s.tokenRegexp.MatchString(token)
	if !matched {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Invalid token.")
		return
	}

	s.writeAuthorizePage(token, "", w, r)
}

func (s *Service) HandleAuthorizeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	username := r.PostFormValue("username")
	password := r.PostFormValue("password")
	token := r.PostFormValue("token")

	if username == "" {
		s.writeAuthorizePage(token, "Missing username.", w, r)
		return
	}
	if password == "" {
		s.writeAuthorizePage(token, "Missing password.", w, r)
		return
	}
	if !s.tokenRegexp.MatchString(token) {
		s.writeAuthorizePage(token, "Invalid token.", w, r)
		return
	}

	userID, err := s.tokens.GetUserID(r.Context(), token)
	if err != nil {
		s.writeAuthorizePage(token, "Invalid token.", w, r)
		return
	}

	_, err = login(r.Context(), username, password)
	if err != nil {
		s.writeAuthorizePage(token, "Invalid credentials.", w, r)
		log.Println(err)
		return
	}

	err = s.creds.SaveCredentials(r.Context(), userID, username, password)
	if err != nil {
		s.writeAuthorizePage(token, "Internal error.", w, r)
		log.Println(err)
		return
	}

	err = s.publisher.PublishEvent(r.Context(), s.credentialsReceivedTopic, nil, userID.String())
	if err != nil {
		s.writeAuthorizePage(token, "Internal error.", w, r)
		log.Println(err)
		return
	}

	err = s.sender.SendNotification(r.Context(), userID, "Otrzymałem Twoje dane logowania.")
	if err != nil {
		log.Println("Couldn't send notification: ", err)
		return
	}

	err = s.tokens.InvalidateAuthorizationToken(r.Context(), token)
	if err != nil {
		log.Println("Couldn't invalidate token: ", err)
	}

	// TODO: Write some success message
}

type signupPageParams struct {
	Token          string
	MessagePresent bool
	Message        string
}

func (s *Service) writeAuthorizePage(token, message string, w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	params := signupPageParams{
		Token:          token,
		MessagePresent: message != "",
		Message:        message,
	}

	err := s.tmpl.Execute(w, params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// TODO: Add links to the description of the app architecture and request the user to accept all terms. Checkboxes maybe
}
