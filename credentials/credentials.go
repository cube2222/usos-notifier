package credentials

import (
	"context"

	"github.com/cube2222/usos-notifier/common/users"
)

type CredentialsStorage interface {
	GetCredentials(ctx context.Context, userID users.UserID) (*Credentials, error)
	SaveCredentials(ctx context.Context, userID users.UserID, user, password string) error
}

type Credentials struct {
	User     string
	Password string
}

type TokenStorage interface {
	GenerateAuthorizationToken(ctx context.Context, userID users.UserID) (string, error)
	GetUserID(ctx context.Context, token string) (users.UserID, error)
	InvalidateAuthorizationToken(ctx context.Context, token string) error
}
