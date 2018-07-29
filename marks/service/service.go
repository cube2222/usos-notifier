package service

// Proposed flow:
// On created: these are the classes you can subscribe to currently. To subscribe send "subscribe <unique prefix>"

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cube2222/usos-notifier/marks"
	"github.com/cube2222/usos-notifier/marks/parser"
	"github.com/pkg/errors"
)

func getScoresForClass(ctx context.Context, classId, session string) (map[string]*marks.Score, error) {
	cli := &http.Client{}

	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"https://usosweb.mimuw.edu.pl/kontroler.php?_action=dla_stud/studia/sprawdziany/pokaz&wez_id=%v",
			classId,
		),
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create request to get scores")
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
		return nil, errors.Wrap(err, "couldn't get scores webpage")
	}
	// TODO: Handle unauthorized = session expired
	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("received non-200 code: %v", res.StatusCode)
	}

	defer res.Body.Close()

	scores, err := parser.GetScores(req.Body)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get scores")
	}

	return scores, nil
}
