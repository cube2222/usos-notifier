package datastore

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/cube2222/usos-notifier/common/users"
	"github.com/cube2222/usos-notifier/credentials"

	"cloud.google.com/go/datastore"
	"github.com/pkg/errors"
	"google.golang.org/api/cloudkms/v1"
)

type encrypted struct {
	UserAndPassword string
}

var encryptionKey = fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
	"usos-notifier", "global", "credentials", "credentials")

type credentialsStorage struct {
	ds  *datastore.Client
	kms *cloudkms.Service
}

func NewCredentialsStorage(ds *datastore.Client, kms *cloudkms.Service) credentials.CredentialsStorage {
	return &credentialsStorage{
		ds:  ds,
		kms: kms,
	}
}

func (cs *credentialsStorage) GetCredentials(ctx context.Context, userID users.UserID) (*credentials.Credentials, error) {
	key := datastore.NameKey("credentials", userID.String(), nil)

	encrypted := encrypted{}

	err := cs.ds.Get(ctx, key, &encrypted)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get encrypted credentials")
	}

	decryptRequest := cloudkms.DecryptRequest{
		AdditionalAuthenticatedData: base64.StdEncoding.EncodeToString([]byte("something")),
		Ciphertext:                  encrypted.UserAndPassword,
	}

	res, err := cs.kms.Projects.Locations.KeyRings.CryptoKeys.
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

	return creds, nil
}

func decodeUserAndPassword(encoded string) (*credentials.Credentials, error) {
	i := strings.Index(encoded, "-")
	if i == -1 {
		return nil, errors.New("missing username length")
	}
	usernameLen, err := strconv.Atoi(encoded[:i])
	if err != nil {
		return nil, errors.Wrap(err, "invalid username length")
	}

	usernameBegin := i + 1
	usernameEnd := usernameBegin + usernameLen

	// +1 because of the dash after the username
	if len(encoded) <= usernameEnd+1 {
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

	return &credentials.Credentials{
		User:     encoded[usernameBegin:usernameEnd],
		Password: passwordPart[passwordBegin:passwordEnd],
	}, nil
}

func (cs *credentialsStorage) SaveCredentials(ctx context.Context, userID users.UserID, user, password string) error {
	credsPhrase := encodeUserAndPassword(user, password)

	encryptRequest := cloudkms.EncryptRequest{
		AdditionalAuthenticatedData: base64.StdEncoding.EncodeToString([]byte("something")), //TODO: Change
		Plaintext:                   base64.StdEncoding.EncodeToString([]byte(credsPhrase)),
	}
	res, err := cs.kms.Projects.Locations.KeyRings.CryptoKeys.Encrypt(encryptionKey, &encryptRequest).Do()
	if err != nil {
		return errors.Wrap(err, "couldn't encrypt credentials")
	}

	key := datastore.NameKey("credentials", userID.String(), nil)

	key, err = cs.ds.Put(ctx, key, &encrypted{
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
