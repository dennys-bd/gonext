package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {
			_, err := db.ExecContext(ctx, `
				CREATE TABLE users (
					id                text PRIMARY KEY,
					email             text NOT NULL UNIQUE,
					password_hash     text NOT NULL,
					role              text NOT NULL,
					email_verified_at timestamptz,
					created_at        timestamptz NOT NULL,
					updated_at        timestamptz NOT NULL
				)
			`)
			if err != nil {
				return err
			}

			_, err = db.ExecContext(ctx, `
				CREATE TABLE sessions (
					id         text PRIMARY KEY,
					user_id    text NOT NULL REFERENCES users (id),
					created_at timestamptz NOT NULL,
					expires_at timestamptz NOT NULL
				)
			`)
			if err != nil {
				return err
			}

			_, err = db.ExecContext(ctx, `
				CREATE TABLE tokens (
					id         text PRIMARY KEY,
					user_id    text NOT NULL REFERENCES users (id),
					kind       text NOT NULL,
					created_at timestamptz NOT NULL,
					expires_at timestamptz NOT NULL,
					used_at    timestamptz
				)
			`)
			if err != nil {
				return err
			}

			_, err = db.ExecContext(ctx, `
				CREATE TABLE permissions (
					id  text PRIMARY KEY,
					key text NOT NULL UNIQUE
				)
			`)
			if err != nil {
				return err
			}

			_, err = db.ExecContext(ctx, `
				CREATE TABLE role_permissions (
					role          text NOT NULL,
					permission_id text NOT NULL REFERENCES permissions (id),
					PRIMARY KEY (role, permission_id)
				)
			`)
			return err
		},
		func(ctx context.Context, db *bun.DB) error {
			_, err := db.ExecContext(ctx, `DROP TABLE role_permissions`)
			if err != nil {
				return err
			}
			_, err = db.ExecContext(ctx, `DROP TABLE permissions`)
			if err != nil {
				return err
			}
			_, err = db.ExecContext(ctx, `DROP TABLE tokens`)
			if err != nil {
				return err
			}
			_, err = db.ExecContext(ctx, `DROP TABLE sessions`)
			if err != nil {
				return err
			}
			_, err = db.ExecContext(ctx, `DROP TABLE users`)
			return err
		},
	)
}
