package notifier

import (
	"context"

	"github.com/cube2222/usos-notifier/common/users"
	"github.com/pkg/errors"
)

type MessengerID string

func NewMessengerID(id string) MessengerID {
	return MessengerID(id)
}

func (id MessengerID) String() string {
	return string(id)
}

type UserMapping interface {
	CreateUser(ctx context.Context, messengerID MessengerID) (users.UserID, error)
	GetMessengerID(ctx context.Context, userID users.UserID) (MessengerID, error)
	GetUserID(ctx context.Context, messengerID MessengerID) (users.UserID, error)
}

var ErrNotFound = errors.New("mapping not found")
