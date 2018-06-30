package tokens

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"cloud.google.com/go/datastore"
	"github.com/pkg/errors"
)

type Tokens struct {
	ds *datastore.Client
}

func NewTokens(cli *datastore.Client) *Tokens {
	return &Tokens{
		ds: cli,
	}
}

type UserID struct {
	UserID string `json:"user_id"`
}

func (t *Tokens) GenerateAuthorizationToken(ctx context.Context, userID string) (string, error) {
	// TODO: Maybe add TTL, isn't that sensitive though.
	n, err := rand.Int(rand.Reader, big.NewInt(1000000000000000))
	if err != nil {
		return "", errors.Wrap(err, "couldn't generate random token")
	}
	token := fmt.Sprintf("%d", n.Int64())

	key := datastore.NameKey("authorization_tokens", token, nil)
	_, err = t.ds.Put(ctx, key, &UserID{userID})
	if err != nil {
		return "", errors.Wrap(err, "couldn't put authorization token into db")
	}

	return token, nil
}

func (t *Tokens) GetUserID(ctx context.Context, token string) (string, error) {
	key := datastore.NameKey("authorization_tokens", token, nil)

	out := UserID{}
	err := t.ds.Get(ctx, key, &out)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get userID")
	}

	return out.UserID, nil
}

func (t *Tokens) InvalidateAuthorizationToken(ctx context.Context, token string) error {
	key := datastore.NameKey("authorization_tokens", token, nil)
	err := t.ds.Delete(ctx, key)
	if err != nil {
		return errors.Wrap(err, "couldn't delete authorization token")
	}
	return nil
}
