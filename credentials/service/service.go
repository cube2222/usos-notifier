package service

import (
	"fmt"
	"html/template"
	"net/http"
	"regexp"

	"github.com/cube2222/grpc-utils/logger"
	"github.com/cube2222/usos-notifier/common/events/publisher"
	"github.com/cube2222/usos-notifier/common/events/subscriber"
	"github.com/cube2222/usos-notifier/common/users"
	"github.com/cube2222/usos-notifier/credentials"
	"github.com/cube2222/usos-notifier/notifier"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type Service struct {
	creds     credentials.CredentialsStorage
	publisher *publisher.Publisher
	tokens    credentials.TokenStorage
	sender    notifier.NotificationSender

	tmpl        *template.Template
	tokenRegexp *regexp.Regexp

	credentialsReceivedTopic string
}

func NewService(credentialsStorage credentials.CredentialsStorage, tokenStorage credentials.TokenStorage, notificationSender notifier.NotificationSender, publisher *publisher.Publisher, authorizationTemplate *template.Template, credentialsReceivedTopic string) (*Service, error) {
	tokenRegexp := regexp.MustCompile("^[0-9]+$")

	service := &Service{
		creds:                    credentialsStorage,
		publisher:                publisher,
		tokens:                   tokenStorage,
		sender:                   notificationSender,
		tmpl:                     authorizationTemplate,
		tokenRegexp:              tokenRegexp,
		credentialsReceivedTopic: credentialsReceivedTopic,
	}

	return service, nil
}

func (s *Service) GetSession(ctx context.Context, r *credentials.GetSessionRequest) (*credentials.GetSessionResponse, error) {
	creds, err := s.creds.GetCredentials(ctx, users.UserID(r.Userid))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get credentials")
	}

	session, err := login(ctx, creds.User, creds.Password)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't login")
	}

	return &credentials.GetSessionResponse{
		Sessionid: session,
	}, nil
}

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

func (s *Service) HandleUserCreatedEvent(ctx context.Context, message *subscriber.Message) error {
	text, err := subscriber.DecodeTextMessage(message)
	if err != nil {
		return subscriber.NewNonRetryableError(errors.Wrap(err, "couldn't decode text message"))
	}

	userID := users.NewUserID(string(text))

	token, err := s.tokens.GenerateAuthorizationToken(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "couldn't generate authorization token")
	}

	// TODO: Add the user id to the token, this way even in the case of duplicate tokens everything will work securely.
	// As the token is a random number up to 10^15 the probability of a repetition with x users is
	// 1 - ( (10^15 - 1)/10^15 * (10^15 - 2)/10^15 ... * (10^15 - x)/10^15 ) < 1 - ( (10^15 - x)/10^15 )^x
	// Which for a million users is less than 10^-3
	// The planned user count is < 100
	// Which is to say, this is if all of them signed up at the same time,
	// as tokens get invalidated after the user provides the credentials.
	// This is a low hanging fruit, but it's also totally unimportant.
	err = s.sender.SendNotification(ctx, userID,
		fmt.Sprintf("Proszę autoryzuj mnie do używania Twoich danych logowania: https://notifier.jacobmartins.com/credentials/authorization?token=%v", token))
	if err != nil {
		return errors.Wrap(err, "couldn't send notification")
	}

	return nil
}
