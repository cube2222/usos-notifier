package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/cube2222/usos-notifier/credentials"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudkms/v1"
	"google.golang.org/api/option"
)

type Service struct {
	ds  *datastore.Client
	kms *cloudkms.Service
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
	ds, err := datastore.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile("C:/Development/Projects/Go/src/github.com/cube2222/usos-notifier/usos-notifier-9a2e44d7f26b.json"))
	if err != nil {
	    return nil, errors.Wrap(err, "couldn't create datastore client")
	}

	return &Service{
		ds:  ds,
		kms: kms,
	}, nil
}

type Encrypted struct {
	UserAndPassword string
}

type Credentials struct {
	User string
	Password string
}

var encryptionKey = fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
	"usos-notifier", "global", "testing", "test-key")

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

	session, err := s.login(creds.User, creds.Password)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't login")
	}

	return &credentials.GetSessionResponse{
		Sessionid: session,
	}, nil
}

func (s *Service) handleSignup(ctx context.Context, user, password, uuid string) error {
	// TODO: Check if you can actually login using this
	credsPhrase := encodeUserAndPassword(user, password)

	encryptRequest := cloudkms.EncryptRequest{
		AdditionalAuthenticatedData: base64.StdEncoding.EncodeToString([]byte("something")),
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

func encodeUserAndPassword(user, password string) string {
	return fmt.Sprintf("%d-%s-%d-%s", len(user), user, len(password), password)
}

func decodeUserAndPassword(encoded string) (*Credentials, error) {
	i := strings.Index(encoded, "-")
	if i == -1 {
		return nil, errors.New("missing username length")
	}
	usernameLen, err := strconv.Atoi(encoded[:i])
	if err != nil {
		return nil, errors.Wrap(err, "invalid username length")
	}

	usernameBegin := i+1
	usernameEnd := usernameBegin + usernameLen

	// +1 because of the dash after the username
	if len(encoded) <= usernameEnd + 1 {
		return nil, errors.New("missing part of encoded username value")
	}

	passwordPart := encoded[usernameEnd+1:]

	i = strings.Index(passwordPart, "-")
	if i == -1 {
		return nil, errors.New("missing password length")
	}
	passwordLen, err := strconv.Atoi(passwordPart[:i])
	if err != nil {
		return nil, errors.Wrap(err, "invalid password length")
	}

	passwordBegin := i + 1
	passwordEnd := passwordBegin + passwordLen

	if len(encoded) <= passwordEnd {
		return nil, errors.New("missing part of encoded password value")
	}

	return &Credentials{
		User:     encoded[usernameBegin:usernameEnd],
		Password: passwordPart[passwordBegin:passwordEnd],
	}, nil
}

func (s *Service) HandleSignupHTTP() func(http.ResponseWriter, *http.Request) {
	type request struct {
		UUID     string `json:"uuid"` // TODO: In the future, just pass the token
		Username string `json:"username"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: add coupon handling

		var creds request

		err := json.NewDecoder(r.Body).Decode(&creds)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, err)
			return
		}

		err = s.handleSignup(r.Context(), creds.Username, creds.Password, creds.UUID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}

		return
	}
}

var ErrAlreadySavedMsg = "Already saved."

func (s *Service) login(user, password string) (string, error) {
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

	resp, err := cli.Get(uri.String())
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
