package datastore

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/cube2222/usos-notifier/common/users"
	"github.com/cube2222/usos-notifier/credentials"

	"cloud.google.com/go/datastore"
	"github.com/pkg/errors"
)

const tokenTable = "authorization_tokens"

type Tokens struct {
	ds *datastore.Client
}

func NewTokenStorage(cli *datastore.Client) credentials.TokenStorage {
	return &Tokens{
		ds: cli,
	}
}

type datastoreUserID struct {
	UserID string `json:"user_id"`
}

func (t *Tokens) GenerateAuthorizationToken(ctx context.Context, userID users.UserID) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000000000000))
	if err != nil {
		return "", errors.Wrap(err, "couldn't generate random token")
	}
	token := fmt.Sprintf("%d", n.Int64())

	key := datastore.NameKey(tokenTable, token, nil)
	_, err = t.ds.Put(ctx, key, &datastoreUserID{userID.String()})
	if err != nil {
		return "", errors.Wrap(err, "couldn't put authorization token into db")
	}

	return token, nil
}

func (t *Tokens) GetUserID(ctx context.Context, token string) (users.UserID, error) {
	key := datastore.NameKey(tokenTable, token, nil)

	out := datastoreUserID{}
	err := t.ds.Get(ctx, key, &out)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get userID")
	}

	return users.NewUserID(out.UserID), nil
}

func (t *Tokens) InvalidateAuthorizationToken(ctx context.Context, token string) error {
	key := datastore.NameKey(tokenTable, token, nil)
	err := t.ds.Delete(ctx, key)
	if err != nil {
		return errors.Wrap(err, "couldn't delete authorization token")
	}
	return nil
}
