package service

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/cube2222/grpc-utils/logger"
	"github.com/cube2222/grpc-utils/requestid"
	"github.com/cube2222/usos-notifier/common/events/publisher"

	"github.com/cube2222/usos-notifier/common/events/subscriber"
	"github.com/cube2222/usos-notifier/common/users"
	"github.com/cube2222/usos-notifier/credentials"
	"github.com/cube2222/usos-notifier/credentials/resources"
	"github.com/cube2222/usos-notifier/credentials/service/datastore"
	"github.com/cube2222/usos-notifier/notifier"

	gdatastore "cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudkms/v1"
	"google.golang.org/api/option"
)

type Service struct {
	creds  credentials.CredentialsStorage
	tokens credentials.TokenStorage
	sender *notifier.NotificationSender

	tmpl        *template.Template
	tokenRegexp *regexp.Regexp
}

func NewService() (*Service, error) {
	// TODO: Create all those outer dependencies in main
	httpCli, err := google.DefaultClient(context.Background(), cloudkms.CloudPlatformScope)
	if err != nil {
		log.Fatal(err)
		return nil, errors.Wrap(err, "couldn't setup google default http client")
	}
	kms, err := cloudkms.New(httpCli)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create cloud kms client")
	}
	ds, err := gdatastore.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create datastore client")
	}
	pubsubCli, err := pubsub.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create datastore client")
	}

	data, err := resources.Asset("authorize.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl, err := template.New("authorize.html").Parse(string(data))
	if err != nil {
		log.Fatal(err)
	}

	tokenRegexp := regexp.MustCompile("^[0-9]+$")

	service := &Service{
		creds:  datastore.NewCredentialsStorage(ds, kms),
		tokens: datastore.NewTokens(ds),
		sender: notifier.NewNotificationSender(
			publisher.
				NewPublisher(pubsubCli).
				Use(publisher.WithRequestID),
		),
		tmpl:        tmpl,
		tokenRegexp: tokenRegexp,
	}

	go func() {
		log.Fatal(
			subscriber.
				NewSubscriptionClient(pubsubCli).
				Subscribe(
					context.Background(),
					"credentials-notifier-user_created",
					subscriber.Chain(
						service.handleUserCreated,
						subscriber.WithLogger(logger.NewStdLogger()),
						subscriber.WithRequestID,
						subscriber.WithLogging(requestid.Key),
					),
				),
		)
	}()

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

func (s *Service) ServeAuthorizationPageHTTP(w http.ResponseWriter, r *http.Request) {
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

type SignupPageParams struct {
	Token          string
	MessagePresent bool
	Message        string
}

func (s *Service) writeAuthorizePage(token, message string, w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	params := SignupPageParams{
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

func (s *Service) handleUserCreated(ctx context.Context, message *subscriber.Message) error {
	text, err := subscriber.DecodeTextMessage(message)
	if err != nil {
		return errors.Wrap(err, "couldn't decode text message")
	}

	userID := users.NewUserID(string(text))

	token, err := s.tokens.GenerateAuthorizationToken(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "couldn't generate authorization token")
	}

	err = s.sender.SendNotification(ctx, userID,
		fmt.Sprintf("Proszę autoryzuj mnie do używania Twoich danych logowania: https://notifier.jacobmartins.com/credentials/authorization?token=%v", token))
	if err != nil {
		return errors.Wrap(err, "couldn't send notification")
	}

	return nil
}
