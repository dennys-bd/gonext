// Package postgres provides the users domain's Postgres-backed
// adapters for its repository, session, and transaction ports.
package postgres

import (
	"context"

	"github.com/uptrace/bun"

	"golden-app/backend/internal/database"
	"golden-app/backend/users/domain"
)

var (
	_ domain.Store    = (*Store)(nil)
	_ domain.TxRunner = (*TxRunner)(nil)
)

// Store is a Postgres-backed domain.Store, built over a bun.IDB so
// the same code runs against the pooled production *bun.DB, against a
// transaction opened by TxRunner, and against a test transaction via
// dbtest.
type Store struct {
	users  *UserRepository
	tokens *TokenRepository
}

// NewStore constructs a Store backed by db.
func NewStore(db bun.IDB) *Store {
	return &Store{users: NewUserRepository(db), tokens: NewTokenRepository(db)}
}

// Users returns the Postgres-backed user repository.
func (s *Store) Users() domain.UserRepository { return s.users }

// Tokens returns the Postgres-backed token repository.
func (s *Store) Tokens() domain.TokenRepository { return s.tokens }

// TxRunner adapts the cross-cutting database.Transactor to the users
// domain's TxRunner port, handing use cases a Store bound to the open
// transaction instead of a raw bun.Tx.
type TxRunner struct {
	transactor database.Transactor
}

// NewTxRunner constructs a TxRunner backed by transactor.
func NewTxRunner(transactor database.Transactor) *TxRunner {
	return &TxRunner{transactor: transactor}
}

// RunInTx runs fn against a Store scoped to a new transaction.
func (r *TxRunner) RunInTx(ctx context.Context, fn func(context.Context, domain.Store) error) error {
	return r.transactor.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return fn(ctx, NewStore(tx))
	})
}
