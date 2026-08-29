package domain

import (
	"context"
	"errors"
	"time"
)

// ErrSessionInvalid is returned for a missing, unknown, or expired
// session token.
var ErrSessionInvalid = errors.New("users: session is invalid")

// Identity is the minimal identity a validated session resolves to —
// enough for a future auth middleware to gate routes by role or
// permission without a second lookup. Permissions is prefetched for
// Role by SessionIssuer.Validate, not queried per HasPermission call.
type Identity struct {
	UserID      string
	Role        string
	Permissions []string
}

// HasRole reports whether the identity's role is exactly role.
func (i Identity) HasRole(role string) bool {
	return i.Role == role
}

// HasPermission reports whether key is among the permissions
// prefetched for the identity's role. A project that never seeds
// role_permissions simply gets false for every key.
func (i Identity) HasPermission(key string) bool {
	for _, p := range i.Permissions {
		if p == key {
			return true
		}
	}
	return false
}

// SessionIssuer issues, validates, and revokes the sessions that back
// a logged-in user. It is deliberately free of any "cookie" or "JWT"
// vocabulary so a future provider (or a JWT implementation for mobile
// clients) can satisfy the same three methods.
//
// Validate returns the full User alongside the Identity so a caller
// that needs to render the account (GET /users/me, the login
// response) does not pay a second round-trip for rows the session
// lookup already had to join against.
type SessionIssuer interface {
	Issue(ctx context.Context, userID string) (token string, expiresAt time.Time, err error)
	Validate(ctx context.Context, token string) (Identity, User, error)
	Revoke(ctx context.Context, token string) error
}
