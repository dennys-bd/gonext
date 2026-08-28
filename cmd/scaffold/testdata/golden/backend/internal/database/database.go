// Package database provides the cross-cutting Postgres connection
// pool shared by every domain: Connect builds a pgx-backed pool,
// wraps it as a database/sql handle, and hands back a *bun.DB for
// domains to query through.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// Connect builds a pgxpool.Pool for dsn, wraps it as a database/sql
// handle via stdlib.OpenDBFromPool, and returns it as a *bun.DB. The
// returned DB's underlying pool is genuinely pgxpool's, so
// Postgres-aware pooling behavior (health checks before a connection
// is handed out, before-acquire/after-release hooks) applies even
// though Bun only sees a standard *sql.DB.
func Connect(ctx context.Context, dsn string) (*bun.DB, error) {
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing database url: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	sqldb := stdlib.OpenDBFromPool(pool)
	db := bun.NewDB(sqldb, pgdialect.New())

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return db, nil
}
