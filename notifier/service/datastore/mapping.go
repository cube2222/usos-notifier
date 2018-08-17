package datastore

import (
	"context"

	"github.com/cube2222/usos-notifier/common/users"
	"github.com/cube2222/usos-notifier/notifier"

	"cloud.google.com/go/datastore"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

const mappingUserIDToMessengerIDTable = "mapping_userid_to_messenger"
const mappingMessengerIDToUserIDTable = "mapping_messenger_to_userid"

type userMapping struct {
	ds *datastore.Client
}

func NewUserMapping(ds *datastore.Client) notifier.UserMapping {
	return &userMapping{
		ds: ds,
	}
}

type datastoreUserID struct {
	UserID string `json:"user_id"`
}

type datastoreMessengerID struct {
	MessengerID string `json:"messenger_id"`
}

func (s *userMapping) CreateUser(ctx context.Context, messengerID notifier.MessengerID) (users.UserID, error) {
	userID, err := uuid.NewV4()
	if err != nil {
		return "", errors.Wrap(err, "couldn't generate uuid")
	}

	tx, err := s.ds.NewTransaction(ctx)
	if err != nil {
		return "", errors.Wrap(err, "couldn't begin transaction")
	}
	defer tx.Rollback()

	key1 := datastore.NameKey(mappingUserIDToMessengerIDTable, userID.String(), nil)
	key2 := datastore.NameKey(mappingMessengerIDToUserIDTable, string(messengerID), nil)

	_, err = tx.Put(key1, &datastoreMessengerID{string(messengerID)})
	if err != nil {
		return "", errors.Wrap(err, "couldn't create userID to messenger mapping")
	}
	_, err = tx.Put(key2, &datastoreUserID{userID.String()})
	if err != nil {
		return "", errors.Wrap(err, "couldn't create messenger to userID mapping")
	}

	_, err = tx.Commit()
	if err != nil {
		return "", errors.Wrap(err, "couldn't commit transaction")
	}

	return users.NewUserID(userID.String()), nil
}

func (s *userMapping) GetMessengerID(ctx context.Context, userID users.UserID) (notifier.MessengerID, error) {
	key := datastore.NameKey(mappingUserIDToMessengerIDTable, string(userID), nil)

	var out datastoreMessengerID
	err := s.ds.Get(ctx, key, &out)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return "", notifier.ErrNotFound
		}
		return "", errors.Wrap(err, "couldn't get messengerID")
	}

	return notifier.NewMessengerID(out.MessengerID), nil
}

func (s *userMapping) GetUserID(ctx context.Context, messengerID notifier.MessengerID) (users.UserID, error) {
	key := datastore.NameKey(mappingMessengerIDToUserIDTable, string(messengerID), nil)

	var out datastoreUserID
	err := s.ds.Get(ctx, key, &out)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return "", notifier.ErrNotFound
		}
		return "", errors.Wrap(err, "couldn't get userID")
	}

	return users.NewUserID(out.UserID), nil
}
