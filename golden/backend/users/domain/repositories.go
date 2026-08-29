package domain

import (
	"context"
	"time"
)

// UserRepository persists and retrieves Users.
type UserRepository interface {
	Create(ctx context.Context, u User) error
	GetByID(ctx context.Context, id string) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	UpdatePasswordHash(ctx context.Context, id, passwordHash string, updatedAt time.Time) error
	MarkEmailVerified(ctx context.Context, id string, verifiedAt time.Time) error
}

// TokenRepository persists and retrieves one-shot Tokens.
type TokenRepository interface {
	Create(ctx context.Context, t Token) error
	Get(ctx context.Context, id string) (Token, error)
	MarkUsed(ctx context.Context, id string, usedAt time.Time) error
}

// Store groups the domain's repositories so a use case can reach both
// through one dependency, and so TxRunner can hand back a set of
// repositories scoped to a single transaction.
type Store interface {
	Users() UserRepository
	Tokens() TokenRepository
}

// TxRunner runs fn against a Store scoped to one transaction,
// committing when fn returns nil and rolling back otherwise. It keeps
// multi-row use cases (registration writes a user and a token) atomic
// without the application layer knowing what a transaction is made of.
type TxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context, s Store) error) error
}

// PasswordHasher hashes and verifies plaintext passwords. The
// algorithm and its cost parameters are an application-layer choice;
// the domain only depends on the operation.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) (bool, error)
}

// Notifier delivers the transactional emails the users domain
// triggers. Core ships a no-op implementation; a real mailer arrives
// with the transactional email feature pack.
type Notifier interface {
	SendConfirmation(ctx context.Context, email, token string) error
	SendAccountExistsNotice(ctx context.Context, email, resetToken string) error
}
