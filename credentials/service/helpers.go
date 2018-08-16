package service

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var ErrAlreadySavedMsg = "Already saved."

// Todo: Remove all those log.Fatals
func login(ctx context.Context, user, password string) (string, error) {
	uri, err := url.Parse("https://logowanie.uw.edu.pl/cas/login")
	if err != nil {
		return "", errors.Wrap(err, "couldn't parse request url")
	}

	q := uri.Query()
	q.Add("service", "https://usosweb.mimuw.edu.pl/kontroler.php?_action=logowaniecas/index")
	q.Add("locale", "pl")
	uri.RawQuery = q.Encode()

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

	req, err := http.NewRequest(http.MethodGet, uri.String(), nil)
	if err != nil {
		log.Fatal(err)
	}

	req = req.WithContext(ctx)

	resp, err := cli.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	LTReg, err := regexp.Compile("LT-[a-zA-Z0-9]+-[a-zA-Z0-9]+")
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Handle inexistance of LT
	LT := LTReg.Find(data)

	form := url.Values{}
	form.Add("username", user)
	form.Add("password", password)
	form.Add("lt", string(LT))
	form.Add("execution", "e1s1")
	form.Add("_eventId", "submit")
	form.Add("submit", "ZALOGUJ")

	// TODO: Identify myself using the UserAgent
	resp, err = cli.PostForm(uri.String(), form)
	if err != nil && !strings.Contains(err.Error(), ErrAlreadySavedMsg) {
		return "", errors.Wrap(err, "couldn't post login form")
	}

	parsed, err := url.Parse("https://usosweb.mimuw.edu.pl")
	if err != nil {
		return "", errors.Wrap(err, "couldn't parse url for cookie extraction")
	}
	for _, cookie := range cli.Jar.Cookies(parsed) {
		if cookie.Name == "PHPSESSID" {
			return cookie.Value, nil
		}
	}
	return "", errors.New("PHPSESSID cookie not found")
}
