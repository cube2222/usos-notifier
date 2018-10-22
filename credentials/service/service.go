package service

import (
	"fmt"
	"html/template"
	"regexp"
	"sync"

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
	sessionCache      map[users.UserID]string
	sessionCacheMutex sync.RWMutex

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
	userid := users.UserID(r.Userid)

	s.sessionCacheMutex.RLock()
	session, ok := s.sessionCache[userid]
	s.sessionCacheMutex.RUnlock()
	if ok {
		logger.FromContext(ctx).Println("Reusing session.")
		return &credentials.GetSessionResponse{
			Sessionid: session,
		}, nil
	}

	creds, err := s.creds.GetCredentials(ctx, userid)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get credentials")
	}

	session, err = login(ctx, creds.User, creds.Password)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't login")
	}

	return &credentials.GetSessionResponse{
		Sessionid: session,
	}, nil
}

func (s *Service) InvalidateSession(ctx context.Context, r *credentials.InvalidateSessionRequest) (*credentials.InvalidateSessionResponse, error) {
	s.sessionCacheMutex.Lock()
	delete(s.sessionCache, users.NewUserID(r.Userid))
	s.sessionCacheMutex.Unlock()

	return &credentials.InvalidateSessionResponse{}, nil
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
