package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cube2222/grpc-utils/logger"
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

func (s *Service) RunScoreChecker(ctx context.Context) {
	for {
		err := s.checkSingleUser(ctx)
		if err != nil {
			if errors.Cause(err) == marks.ErrNoUserToCheck {
				time.Sleep(time.Minute)
				continue
			}
			logger.FromContext(ctx).Println(err)
			continue
		}
	}
}

func (s *Service) checkSingleUser(ctx context.Context) error {
	userID, user, err := s.users.HandleNextCheck(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't get next user to check")
	}

	session, err := s.getSession(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "couldn't get session")
	}

	updatedUser, err := getUpdatedUser(ctx, session, user)
	if err != nil {
		return errors.Wrap(err, "couldn't get updated scores")
	}

	changes := getChangedScores(user, updatedUser)
	if len(changes) == 0 {
		return nil
	}

	for class, scores := range changes {
		lines := make([]string, len(scores)+1)
		lines[0] = fmt.Sprintf("New scores have appeared in %s:", class)
		for i, score := range scores {
			lines[i+1] = fmt.Sprintf("%s: %v/%v", score.Name, score.Actual, score.Max)
		}

		err = s.sender.SendNotification(ctx, userID, strings.Join(lines, "\n"))
		if err != nil {
			return errors.Wrap(err, "couldn't send notification")
		}
	}

	err = s.users.Set(ctx, userID, updatedUser)
	if err != nil {
		return errors.Wrap(err, "couldn't save user")
	}

	return nil
}

func getUpdatedUser(ctx context.Context, session string, user *marks.User) (*marks.User, error) {
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

		out.Classes = append(out.Classes, parser.MakeClassWithScores(class.ID, class.Name, scores))
	}

	sort.Slice(out.Classes, func(i, j int) bool {
		return out.Classes[i].ID < out.Classes[j].ID
	})

	return out, nil
}

func getChangedScores(old *marks.User, new *marks.User) map[string][]marks.Score {
	changed := make(map[string][]marks.Score)
	oldScores := make(map[string]map[string]marks.Score)
	for _, class := range old.Classes {
		oldScores[class.ID] = make(map[string]marks.Score)
		for _, score := range class.Scores {
			oldScores[class.ID][score.Name] = score
		}
	}

	for _, class := range new.Classes {
		for _, newScore := range class.Scores {
			oldScore, ok := oldScores[class.ID][newScore.Name]

			newVisibleScore := !ok && !newScore.Unknown && !newScore.Hidden
			scoreRevealed := ok && !oldScore.Visible() && newScore.Visible()
			scoreUpdated := ok && oldScore.Visible() && newScore.Visible() && oldScore.Actual != newScore.Actual

			if newVisibleScore || scoreRevealed || scoreUpdated {
				changed[class.Name] = append(changed[class.Name], newScore)
			}
		}
	}

	return changed
}
