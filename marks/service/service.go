package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/cube2222/usos-notifier/common/events/subscriber"
	"github.com/cube2222/usos-notifier/common/users"
	"github.com/cube2222/usos-notifier/credentials"
	"github.com/cube2222/usos-notifier/marks"
	"github.com/cube2222/usos-notifier/marks/parser"
	"github.com/cube2222/usos-notifier/notifier"
	"github.com/pkg/errors"
)

var ErrSessionExpired = errors.New("session expired")

type Service struct {
	credentials credentials.CredentialsClient
	sender      notifier.NotificationSender
	users       marks.UserStorage
}

func NewService(credentials credentials.CredentialsClient, sender notifier.NotificationSender, users marks.UserStorage) *Service {
	return &Service{
		credentials: credentials,
		sender:      sender,
		users:       users,
	}
}

func (s *Service) HandleCredentialsProvidedEvent(ctx context.Context, message *subscriber.Message) error {
	text, err := subscriber.DecodeTextMessage(message)
	if err != nil {
		return subscriber.NewNonRetryableError(errors.Wrap(err, "couldn't decode text message"))
	}

	userID := users.NewUserID(string(text))

	res, err := s.credentials.GetSession(ctx, &credentials.GetSessionRequest{
		Userid: userID.String(),
	})
	if err != nil {
		return errors.Wrap(err, "couldn't get session")
	}
	session := res.Sessionid

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
		ObservedClasses: user.ObservedClasses,
		Classes:         make([]marks.Class, len(user.ObservedClasses)),
		NextCheck:       user.NextCheck,
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

func getAuthorizedWebsite(ctx context.Context, httpCli *http.Client, session, path string) (io.ReadCloser, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s%s", "https://usosweb.mimuw.edu.pl", path),
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create authorized request")
	}

	//req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.92 Safari/537.36")

	req.AddCookie(
		&http.Cookie{
			Name:     "PHPSESSID",
			Value:    session,
			Path:     "/",
			Domain:   "usosweb.mimuw.edu.pl",
			HttpOnly: true,

			Expires: time.Now().Add(time.Minute * 15),
		},
	)

	req = req.WithContext(ctx)

	res, err := httpCli.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get authorized website")
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		if res.StatusCode == http.StatusForbidden {
			return nil, ErrSessionExpired
		}
		return nil, errors.Errorf("received non-200 code: %v", res.StatusCode)
	}

	return res.Body, nil
}

func getClasses(ctx context.Context, httpCli *http.Client, session string) (map[string]*parser.Class, error) {
	body, err := getAuthorizedWebsite(ctx, httpCli, session, "/kontroler.php?_action=home/index")
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get authorized website")
	}
	defer body.Close()

	classes, err := parser.GetClasses(body)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get classes")
	}

	return classes, nil
}

func getScoresForClass(ctx context.Context, httpCli *http.Client, session, classId string) (map[string]*parser.Score, error) {
	body, err := getAuthorizedWebsite(
		ctx,
		httpCli,
		session,
		fmt.Sprintf(
			"/kontroler.php?_action=dla_stud/studia/sprawdziany/pokaz&wez_id=%v",
			classId,
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get authorized website")
	}
	defer body.Close()

	scores, err := parser.GetScores(body)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get scores")
	}

	return scores, nil
}
