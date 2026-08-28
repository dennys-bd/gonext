package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
)

// Transactor runs fn inside a database transaction, committing if fn
// returns nil and rolling back otherwise. Nothing in this phase needs
// a multi-write atomic operation, but the seam is built now so a
// future transactional service can depend on Transactor instead of a
// concrete *bun.DB from day one, and be tested via dbtest's
// savepoint-based implementation.
type Transactor interface {
	RunInTx(ctx context.Context, opts *sql.TxOptions, fn func(context.Context, bun.Tx) error) error
}

var _ Transactor = (*BunTransactor)(nil)

// BunTransactor is the production Transactor, backed by a real
// top-level *bun.DB transaction.
type BunTransactor struct {
	db *bun.DB
}

// NewBunTransactor constructs a BunTransactor backed by db.
func NewBunTransactor(db *bun.DB) *BunTransactor {
	return &BunTransactor{db: db}
}

// RunInTx runs fn inside a real transaction opened on t's underlying
// *bun.DB.
func (t *BunTransactor) RunInTx(ctx context.Context, opts *sql.TxOptions, fn func(context.Context, bun.Tx) error) error {
	if err := t.db.RunInTx(ctx, opts, fn); err != nil {
		return fmt.Errorf("running transaction: %w", err)
	}
	return nil
}
