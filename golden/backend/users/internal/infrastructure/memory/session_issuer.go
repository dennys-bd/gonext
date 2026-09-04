package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dennys-bd/gonext/auth"

	"golden-app/backend/users/domain"
)

var _ domain.SessionIssuer = (*SessionIssuer)(nil)

// SessionIssuer is an in-memory domain.SessionIssuer. It resolves
// Identity from the users it shares with store, so a test can seed a
// user through the repository and immediately validate a session for
// it. Permissions are always empty: role_permissions is a
// Postgres-only join, and every use case treats an empty slice as
// valid.
type SessionIssuer struct {
	mu       sync.Mutex
	users    domain.UserRepository
	ttl      time.Duration
	seq      int
	sessions map[string]session
}

type session struct {
	userID    string
	expiresAt time.Time
}

// NewSessionIssuer constructs an in-memory SessionIssuer resolving
// identities against users.
func NewSessionIssuer(users domain.UserRepository, ttl time.Duration) *SessionIssuer {
	return &SessionIssuer{users: users, ttl: ttl, sessions: make(map[string]session)}
}

// Issue mints a predictable session token for userID.
func (i *SessionIssuer) Issue(_ context.Context, userID string) (string, time.Time, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.seq++
	token := fmt.Sprintf("session-%d", i.seq)
	expiresAt := time.Now().UTC().Add(i.ttl)
	i.sessions[token] = session{userID: userID, expiresAt: expiresAt}
	return token, expiresAt, nil
}

// Validate resolves token to an Identity and the account behind it,
// or domain.ErrSessionInvalid.
func (i *SessionIssuer) Validate(ctx context.Context, token string) (auth.Identity, domain.User, error) {
	i.mu.Lock()
	s, ok := i.sessions[token]
	i.mu.Unlock()
	if !ok || !time.Now().UTC().Before(s.expiresAt) {
		return auth.Identity{}, domain.User{}, domain.ErrSessionInvalid
	}

	u, err := i.users.GetByID(ctx, s.userID)
	if err != nil {
		return auth.Identity{}, domain.User{}, domain.ErrSessionInvalid
	}
	return auth.Identity{UserID: u.ID, Role: u.Role, Permissions: []string{}}, u, nil
}

// Revoke forgets token. Revoking an unknown token is not an error, so
// logging out twice behaves the same as logging out once.
func (i *SessionIssuer) Revoke(_ context.Context, token string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	delete(i.sessions, token)
	return nil
}
