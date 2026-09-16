package migrations

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/dennys-bd/gonext/dbmigrate"
)

func init() {
	dbmigrate.Register(
		func(ctx context.Context, db *bun.DB) error {
			_, err := db.ExecContext(ctx, `
				CREATE TABLE categories (
					id         text PRIMARY KEY,
					name       text NOT NULL,
					created_at timestamptz NOT NULL,
					updated_at timestamptz NOT NULL
				)
			`)
			return err
		},
		func(ctx context.Context, db *bun.DB) error {
			_, err := db.ExecContext(ctx, `DROP TABLE categories`)
			return err
		},
	)
}
