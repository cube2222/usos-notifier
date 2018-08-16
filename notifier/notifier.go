package notifier

import "context"

type UserID string

func NewUserID(id string) UserID {
	return UserID(id)
}

func (id UserID) String() string {
	return string(id)
}

type MessengerID string

func NewMessengerID(id string) MessengerID {
	return MessengerID(id)
}

func (id MessengerID) String() string {
	return string(id)
}

type UserMapping interface {
	CreateUser(ctx context.Context, messengerID MessengerID) (UserID, error)
	GetMessengerID(ctx context.Context, userID UserID) (MessengerID, error)
	GetUserID(ctx context.Context, messengerID MessengerID) (UserID, error)
}
