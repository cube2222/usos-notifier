package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/cube2222/usos-notifier/common/events/subscriber"
	"github.com/cube2222/usos-notifier/common/users"
	"github.com/cube2222/usos-notifier/credentials"
	"github.com/cube2222/usos-notifier/marks"
	"github.com/cube2222/usos-notifier/marks/parser"
	"github.com/cube2222/usos-notifier/notifier"
	"github.com/cube2222/usos-notifier/notifier/commands"
	"github.com/pkg/errors"
)

var ErrSessionExpired = errors.New("session expired")

type Service struct {
	commandsHandler commands.CommandsHandler
	credentials     credentials.CredentialsClient
	sender          notifier.NotificationSender
	users           marks.UserStorage
}

func NewService(credentials credentials.CredentialsClient, sender notifier.NotificationSender, users marks.UserStorage) *Service {
	s := &Service{
		commandsHandler: commands.NewCommandsHandler(sender),
		credentials:     credentials,
		sender:          sender,
		users:           users,
	}

	s.commandsHandler.Handle(commands.RegexpMatcher(regexp.MustCompile("^[Ss]ubscribe to (?P<class_id>.+)$")), s.SubscribeClass)
	s.commandsHandler.Handle(commands.RegexpMatcher(regexp.MustCompile("^[Uu]nsubscribe from (?P<class_id>.+)$")), s.UnsubscribeClass)
	s.commandsHandler.Handle(commands.RegexpMatcher(regexp.MustCompile("^[Ll]ist( classes)?$")), s.ListClasses)

	return s
}

func (s *Service) HandleUserMessageEvent(ctx context.Context, message *subscriber.Message) error {
	return s.commandsHandler.HandleMessage(ctx, message)
}

func (s *Service) getSession(ctx context.Context, userID users.UserID) (string, error) {
	res, err := s.credentials.GetSession(ctx, &credentials.GetSessionRequest{
		Userid: userID.String(),
	})
	if err != nil {
		return "", errors.Wrap(err, "couldn't get session from credentials service")
	}

	return res.Sessionid, nil
}

func (s *Service) HandleCredentialsProvidedEvent(ctx context.Context, message *subscriber.Message) error {
	text, err := subscriber.DecodeTextMessage(message)
	if err != nil {
		return subscriber.NewNonRetryableError(errors.Wrap(err, "couldn't decode text message"))
	}

	userID := users.NewUserID(string(text))

	session, err := s.getSession(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "couldn't get session")
	}

	user, err := initializeUser(ctx, session)
	if err != nil {
		return errors.Wrap(err, "couldn't get user")
	}

	err = s.users.Set(ctx, userID, user)
	if err != nil {
		return errors.Wrap(err, "couldn't save user")
	}

	lines := make([]string, len(user.AvailableClasses)+1)
	lines[0] = "These are the classes you can subscribe to:"
	for i, class := range user.AvailableClasses {
		lines[i+1] = fmt.Sprintf("%v: %v", class.ID, class.Name)
	}

	err = s.sender.SendNotification(ctx, userID, strings.Join(lines, "\n"))
	if err != nil {
		return errors.Wrap(err, "couldn't send notification")
	}

	return nil
}

func initializeUser(ctx context.Context, session string) (*marks.User, error) {
	cli := &http.Client{}
	out := &marks.User{}

	classes, err := getClasses(ctx, cli, session)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get classes")
	}

	for id, class := range classes {
		out.AvailableClasses = append(out.AvailableClasses, marks.ClassHeader{
			ID:   id,
			Name: class.Name,
		})
	}

	return out, nil
}

func updateScores(ctx context.Context, session string, user *marks.User) (*marks.User, error) {
	cli := &http.Client{}

	out := &marks.User{
		AvailableClasses: user.AvailableClasses,
		ObservedClasses:  user.ObservedClasses,
		Classes:          make([]marks.Class, len(user.ObservedClasses)),
		NextCheck:        user.NextCheck,
	}

	for _, class := range out.ObservedClasses {
		scores, err := getScoresForClass(ctx, cli, session, class.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't get scores for class %v", class.ID)
		}

		user.Classes = append(user.Classes, parser.MakeClassWithScores(class.ID, class.Name, scores))
	}

	sort.Slice(user.Classes, func(i, j int) bool {
		return user.Classes[i].ID < user.Classes[j].ID
	})

	return out, nil
}
