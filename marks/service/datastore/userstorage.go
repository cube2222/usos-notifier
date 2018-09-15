package datastore

import (
	"context"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/cube2222/usos-notifier/common/users"
	"github.com/cube2222/usos-notifier/marks"
	"github.com/pkg/errors"
)

type userStorage struct {
	ds *datastore.Client
}

func NewUserStorage(ds *datastore.Client) marks.UserStorage {
	return &userStorage{
		ds: ds,
	}
}

func (s *userStorage) Get(ctx context.Context, userID users.UserID) (*marks.User, error) {
	out := &marks.User{}

	key := datastore.NameKey("scores", userID.String(), nil)
	err := s.ds.Get(ctx, key, out)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, marks.ErrUserNotFound
		}
		return nil, err
	}
	return out, nil
}

func (s *userStorage) Set(ctx context.Context, userID users.UserID, user *marks.User) (err error) {
	key := datastore.NameKey("scores", userID.String(), nil)
	_, err = s.ds.Put(ctx, key, user)
	return
}

func (s *userStorage) HandleNextCheck(ctx context.Context) (users.UserID, *marks.User, error) {
	var out []*marks.User

	query := datastore.NewQuery("scores").Filter("NextCheck <", time.Now()).Limit(1)
	keys, err := s.ds.GetAll(context.Background(), query, &out)
	if err != nil {
		return "", nil, errors.Wrap(err, "couldn't get user for query")
	}
	if len(keys) == 0 {
		return "", nil, marks.ErrNoUserToCheck
	}

	tx, err := s.ds.NewTransaction(ctx)
	if err != nil {
		return "", nil, errors.Wrap(err, "couldn't begin transaction")
	}
	defer tx.Rollback()

	err = s.ds.Get(ctx, keys[0], out[0])
	if err != nil {
		return "", nil, errors.Wrap(err, "couldn't get user to check if still eligible")
	}
	if out[0].NextCheck.After(time.Now()) {
		// This is left so intentionally at this moment.
		// If it gets to be a problem, we'll fix it.
		return "", nil, errors.Wrap(err, "user no more eligible")
	}

	out[0].NextCheck = time.Now().Add(15 * time.Minute) // TODO: Make configurable
	_, err = tx.Put(keys[0], out[0])
	if err != nil {
		return "", nil, errors.Wrap(err, "couldn't set next check")
	}

	_, err = tx.Commit()
	if err != nil {
		return "", nil, errors.Wrap(err, "couldn't commit transaction")
	}
	return users.NewUserID(keys[0].Name), out[0], nil
}
