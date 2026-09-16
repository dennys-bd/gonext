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
				CREATE TABLE widgets (
					id         text PRIMARY KEY,
					name       text NOT NULL,
					count      integer NOT NULL,
					total      bigint NOT NULL,
					ratio      double precision NOT NULL,
					active     boolean NOT NULL,
					due        timestamptz NOT NULL,
					created_at timestamptz NOT NULL,
					updated_at timestamptz NOT NULL
				)
			`)
			return err
		},
		func(ctx context.Context, db *bun.DB) error {
			_, err := db.ExecContext(ctx, `DROP TABLE widgets`)
			return err
		},
	)
}
