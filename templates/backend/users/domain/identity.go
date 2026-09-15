package domain

import (
	"context"
	"errors"
	"time"

	"github.com/dennys-bd/gonext/auth"
)

// ErrSessionInvalid is returned for a missing, unknown, or expired session token.
var ErrSessionInvalid = errors.New("users: session is invalid")

// SessionIssuer issues, validates, and revokes the sessions that back a logged-in user.
// Validate returns the full User alongside the auth.Identity to avoid a second lookup.
type SessionIssuer interface {
	Issue(ctx context.Context, userID string) (token string, expiresAt time.Time, err error)
	Validate(ctx context.Context, token string) (auth.Identity, User, error)
	Revoke(ctx context.Context, token string) error
}
