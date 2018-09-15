package service

// Proposed flow:
// On created: these are the classes you can subscribe to currently. To subscribe send "subscribe <unique prefix>"

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cube2222/usos-notifier/marks"
	"github.com/cube2222/usos-notifier/marks/parser"
	"github.com/pkg/errors"
)

var ErrSessionExpired = errors.New("session expired")

func getAuthorizedWebsite(ctx context.Context, session, path string) (io.ReadCloser, error) {
	cli := &http.Client{}

	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s%s", "https://usosweb.mimuw.edu.pl", path),
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create authorized request")
	}

	req.AddCookie(
		&http.Cookie{
			Name:    "PHPSESSID",
			Value:   session,
			Path:    "/",
			Domain:  "usosweb.mimuw.edu.pl",
			Expires: time.Now().Add(time.Minute * 15),
		},
	)

	req = req.WithContext(ctx)

	res, err := cli.Do(req)
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

func getClasses(ctx context.Context, session, url string) (map[string]*marks.Class, error) {
	body, err := getAuthorizedWebsite(ctx, session, "/kontroler.php?_action=home/index")
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

func getScoresForClass(ctx context.Context, classId, session string) (map[string]*marks.Score, error) {
	body, err := getAuthorizedWebsite(
		ctx,
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
