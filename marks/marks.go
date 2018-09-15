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
	ObservedClasses []string
	Classes         []Class
	NextCheck       time.Time
}

type Class struct {
	ID     string
	Name   string
	Scores []Score
}

type Score struct {
	Name        string
	Unknown     bool
	Hidden      bool
	Actual, Max float64
}

type UserStorage interface {
	Get(ctx context.Context, userID users.UserID) (*User, error)
	Set(ctx context.Context, userID users.UserID, user *User) error
	HandleNextCheck(ctx context.Context) (users.UserID, *User, error)
}
