package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cube2222/usos-notifier/marks/parser"
	"github.com/pkg/errors"
)

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
