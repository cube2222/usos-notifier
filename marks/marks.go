package marks

import (
	"context"
	"errors"
	"time"

	"github.com/cube2222/usos-notifier/common/users"
)

var ErrUserNotFound = errors.New("user not found")
var ErrNoUserToCheck = errors.New("no user to check")

type User struct {
	AvailableClasses []ClassHeader
	ObservedClasses  []ClassHeader
	Classes          []Class
	NextCheck        time.Time
}

type ClassHeader struct {
	ID   string
	Name string
}

type Class struct {
	ClassHeader
	Scores []Score
}

type Score struct {
	Name        string
	Unknown     bool
	Hidden      bool
	Actual, Max float64
}

func (s *Score) Visible() bool {
	return !s.Unknown && !s.Hidden
}

type UserStorage interface {
	Get(ctx context.Context, userID users.UserID) (*User, error)
	Set(ctx context.Context, userID users.UserID, user *User) error
	HandleNextCheck(ctx context.Context) (users.UserID, *User, error)
}
