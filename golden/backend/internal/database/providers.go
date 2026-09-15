package database

import (
	"context"

	"github.com/uptrace/bun"
)

// ProvideDB wraps Connect for wire, pairing the *bun.DB with a cleanup
// closure that closes it, for wire's generated cleanup chain.
func ProvideDB(ctx context.Context, dsn string) (*bun.DB, func(), error) {
	db, err := Connect(ctx, dsn)
	if err != nil {
		return nil, nil, err
	}
	return db, func() { _ = db.Close() }, nil
}
