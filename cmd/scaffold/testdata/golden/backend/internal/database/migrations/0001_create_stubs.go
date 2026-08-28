package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {
			_, err := db.ExecContext(ctx, `
				CREATE TABLE stubs (
					id         text PRIMARY KEY,
					name       text NOT NULL,
					created_at timestamptz NOT NULL
				)
			`)
			return err
		},
		func(ctx context.Context, db *bun.DB) error {
			_, err := db.ExecContext(ctx, `DROP TABLE stubs`)
			return err
		},
	)
}
