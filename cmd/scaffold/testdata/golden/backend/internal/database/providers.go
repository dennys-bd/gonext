package database

import (
	"context"

	"github.com/uptrace/bun"
)

// ProvideDB wraps Connect for wire: it pairs the *bun.DB with a
// cleanup closure that closes it, so the generated injector's cleanup
// chain closes the pool automatically instead of main.go managing a
// manual defer db.Close().
func ProvideDB(ctx context.Context, dsn string) (*bun.DB, func(), error) {
	db, err := Connect(ctx, dsn)
	if err != nil {
		return nil, nil, err
	}
	return db, func() { _ = db.Close() }, nil
}
