package service

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var ErrAlreadySavedMsg = "Already saved."

var ltRegexp = regexp.MustCompile("LT-[a-zA-Z0-9]+-[a-zA-Z0-9]+")

func login(ctx context.Context, user, password string) (string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", errors.Wrap(err, "couldn't create empty cookiejar")
	}

	redir := ""
	// Better create a new http client each time.
	// We explicitly don't want to share any state.
	cli := http.Client{
		Jar:     jar,
		Timeout: time.Second * 40,

		// This will make sure we only get redirected once and fulfill the ticket exchange
		// USOS tends to throw us into an infinite redirection loop.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if redir == "" {
				redir = req.URL.String()
				return nil
			} else {
				return errors.New(ErrAlreadySavedMsg)
			}
		},
	}

	uri, err := url.Parse("https://logowanie.uw.edu.pl/cas/login")
	if err != nil {
		return "", errors.Wrap(err, "couldn't parse request url")
	}

	q := uri.Query()
	q.Add("service", "https://usosweb.mimuw.edu.pl/kontroler.php?_action=logowaniecas/index")
	q.Add("locale", "pl")
	uri.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, uri.String(), nil)
	if err != nil {
		return "", errors.Wrap(err, "couldn't create login page get request")
	}

	req = req.WithContext(ctx)

	resp, err := cli.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get login page")
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "couldn't read login page body")
	}

	LT := ltRegexp.Find(data)
	if len(LT) == 0 {
		return "", errors.Wrap(err, "couldn't retrieve login token from the login page body")
	}

	form := url.Values{}
	form.Add("username", user)
	form.Add("password", password)
	form.Add("lt", string(LT))
	form.Add("execution", "e1s1")
	form.Add("_eventId", "submit")
	form.Add("submit", "ZALOGUJ")

	// TODO: Identify myself using the UserAgent
	resp, err = cli.PostForm(uri.String(), form)
	// USOS throws us into an infinite redirection loop. So we're breaking
	// after the first redirect which provided us with the USOS session token.
	if err != nil && !strings.Contains(err.Error(), ErrAlreadySavedMsg) {
		return "", errors.Wrap(err, "couldn't login")
	}

	parsed, err := url.Parse("https://usosweb.mimuw.edu.pl")
	if err != nil {
		return "", errors.Wrap(err, "couldn't parse url for usos session cookie extraction")
	}
	for _, cookie := range cli.Jar.Cookies(parsed) {
		if cookie.Name == "PHPSESSID" {
			return cookie.Value, nil
		}
	}
	return "", errors.New("usos session cookie not found")
}
