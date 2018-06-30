package service

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"github.com/cube2222/usos-notifier/common/events"
	"github.com/cube2222/usos-notifier/credentials"
	"github.com/cube2222/usos-notifier/credentials/resources"
	"github.com/cube2222/usos-notifier/credentials/service/tokens"
	"github.com/cube2222/usos-notifier/notifier"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudkms/v1"
	"google.golang.org/api/option"
)

type Service struct {
	ds  *datastore.Client
	kms *cloudkms.Service

	tokens *tokens.Tokens
	sender *notifier.NotificationSender

	tmpl        *template.Template
	tokenRegexp *regexp.Regexp
}

func NewService() (*Service, error) {
	cli, err := google.DefaultClient(context.Background(), cloudkms.CloudPlatformScope)
	if err != nil {
		log.Fatal(err)
		return nil, errors.Wrap(err, "couldn't setup google default http client")
	}

	kms, err := cloudkms.New(cli)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create cloud kms client")
	}
	ds, err := datastore.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create datastore client")
	}
	pubsub, err := pubsub.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create datastore client")
	}

	tokens := tokens.NewTokens(ds)

	sender := notifier.NewNotificationSender(pubsub)

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
		ds:          ds,
		kms:         kms,
		tokens:      tokens,
		sender:      sender,
		tmpl:        tmpl,
		tokenRegexp: tokenRegexp,
	}

	go func() {
		log.Fatal(pubsub.Subscription("credentials-notifier-user_created").Receive(context.Background(), service.handleUserCreated))
	}()

	return service, nil
}

type Encrypted struct {
	UserAndPassword string
}

var encryptionKey = fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
	"usos-notifier", "global", "credentials", "credentials")

// TODO: Errors should use grpc error code package
func (s *Service) GetSession(ctx context.Context, r *credentials.GetSessionRequest) (*credentials.GetSessionResponse, error) {
	key := datastore.NameKey("credentials", r.Userid, nil)

	encrypted := Encrypted{}

	err := s.ds.Get(ctx, key, &encrypted)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get encrypted credentials")
	}

	decryptRequest := cloudkms.DecryptRequest{
		AdditionalAuthenticatedData: base64.StdEncoding.EncodeToString([]byte("something")),
		Ciphertext:                  encrypted.UserAndPassword,
	}

	res, err := s.kms.Projects.Locations.KeyRings.CryptoKeys.
		Decrypt(encryptionKey, &decryptRequest).
		Context(ctx).
		Do()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't decrypt credentials")
	}

	decrypted, err := base64.StdEncoding.DecodeString(res.Plaintext)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't base64 decode credentials")
	}

	creds, err := decodeUserAndPassword(string(decrypted))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't decode credentials")
	}

	session, err := login(creds.User, creds.Password)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't login")
	}

	return &credentials.GetSessionResponse{
		Sessionid: session,
	}, nil
}

func (s *Service) handleSignup(ctx context.Context, user, password, uuid string) error {
	// TODO: Check if you can actually login using this before accepting
	credsPhrase := encodeUserAndPassword(user, password)

	encryptRequest := cloudkms.EncryptRequest{
		AdditionalAuthenticatedData: base64.StdEncoding.EncodeToString([]byte("something")), //TODO: Change
		Plaintext:                   base64.StdEncoding.EncodeToString([]byte(credsPhrase)),
	}
	res, err := s.kms.Projects.Locations.KeyRings.CryptoKeys.Encrypt(encryptionKey, &encryptRequest).Do()
	if err != nil {
		return errors.Wrap(err, "couldn't encrypt credentials")
	}

	key := datastore.NameKey("credentials", uuid, nil)

	key, err = s.ds.Put(ctx, key, &Encrypted{
		UserAndPassword: res.Ciphertext,
	})
	if err != nil {
		return errors.Wrap(err, "couldn't save credentials")
	}

	return nil
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

	err = s.handleSignup(r.Context(), username, password, userID)
	// TODO: Better error information
	if err != nil {
		s.writeAuthorizePage(token, "Error.", w, r)
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

func (s *Service) handleUserCreated(ctx context.Context, message *pubsub.Message) {
	defer message.Nack()

	data, err := events.DecodeTextMessage(message)
	if err != nil {
		log.Println("Couldn't decode text message: ", err)
		return
	}

	userID := string(data)

	token, err := s.tokens.GenerateAuthorizationToken(ctx, userID)
	if err != nil {
		log.Println("Couldn't generate authorization token: ", err)
		return
	}

	err = s.sender.SendNotification(ctx, userID,
		fmt.Sprintf("Proszę autoryzuj mnie do używania Twoich danych logowania: https://notifier.jacobmartins.com/credentials/authorization?token=%v", token))
	if err != nil {
		log.Println("Couldn't send notification: ", err)
		return
	}

	message.Ack()
}
