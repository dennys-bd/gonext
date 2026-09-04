package domain

import (
	"context"
	"errors"
	"time"

	"github.com/dennys-bd/gonext/auth"
)

// ErrSessionInvalid is returned for a missing, unknown, or expired
// session token. It stays here rather than moving to the published
// contract because it describes this track's sessions; the Resolver
// adapter is what translates it to auth.ErrUnauthenticated.
var ErrSessionInvalid = errors.New("users: session is invalid")

// SessionIssuer issues, validates, and revokes the sessions that back
// a logged-in user. It is deliberately free of any "cookie" or "JWT"
// vocabulary so a JWT implementation for mobile clients can satisfy
// the same three methods.
//
// Validate returns the full User alongside the auth.Identity so a
// caller that needs to render the account does not pay a second
// round-trip for rows the session lookup already joined against.
type SessionIssuer interface {
	Issue(ctx context.Context, userID string) (token string, expiresAt time.Time, err error)
	Validate(ctx context.Context, token string) (auth.Identity, User, error)
	Revoke(ctx context.Context, token string) error
}
