package mapping

import (
	"context"

	"cloud.google.com/go/datastore"
	"github.com/cube2222/usos-notifier/notifier"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

type datastoreUserMapping struct {
	ds *datastore.Client
}

func NewDatastoreUserMapping(ds *datastore.Client) notifier.UserMapping {
	return &datastoreUserMapping{
		ds: ds,
	}
}

type datastoreUserID struct {
	UserID string `json:"user_id"`
}

type datastoreMessengerID struct {
	MessengerID string `json:"messenger_id"`
}

func (s *datastoreUserMapping) CreateUser(ctx context.Context, messengerID notifier.MessengerID) (notifier.UserID, error) {
	userID, err := uuid.NewV4()
	if err != nil {
		return "", errors.Wrap(err, "couldn't generate uuid")
	}

	tx, err := s.ds.NewTransaction(ctx)
	if err != nil {
		return "", errors.Wrap(err, "couldn't begin transaction")
	}
	defer tx.Rollback()

	key1 := datastore.NameKey("mapping-userid-to-messenger", userID.String(), nil)
	key2 := datastore.NameKey("mapping-messenger-to-userid", string(messengerID), nil)

	_, err = tx.Put(key1, &datastoreMessengerID{string(messengerID)})
	if err != nil {
		return "", errors.Wrap(err, "couldn't create userid to messenger mapping")
	}
	_, err = tx.Put(key2, &datastoreUserID{userID.String()})
	if err != nil {
		return "", errors.Wrap(err, "couldn't create messenger to userid mapping")
	}

	_, err = tx.Commit()
	if err != nil {
		return "", errors.Wrap(err, "couldn't commit transaction")
	}

	return notifier.NewUserID(userID.String()), nil
}

func (s *datastoreUserMapping) GetMessengerID(ctx context.Context, userID notifier.UserID) (notifier.MessengerID, error) {
	key := datastore.NameKey("mapping-userid-to-messenger", string(userID), nil)

	var out datastoreMessengerID
	err := s.ds.Get(ctx, key, &out)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get messengerID")
	}

	return notifier.NewMessengerID(out.MessengerID), nil
}

func (s *datastoreUserMapping) GetUserID(ctx context.Context, messengerID notifier.MessengerID) (notifier.UserID, error) {
	key := datastore.NameKey("mapping-messenger-to-userid", string(messengerID), nil)

	var out datastoreUserID
	err := s.ds.Get(ctx, key, &out)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get userID")
	}

	return notifier.NewUserID(out.UserID), nil
}
