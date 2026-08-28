package database

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/uptrace/bun"
)

func TestBunTransactor_RunInTx(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; run `make db-up` and export it to run this test")
	}

	db, err := Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS transactor_scratch (id int PRIMARY KEY)`); err != nil {
		t.Fatalf("creating scratch table: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DROP TABLE IF EXISTS transactor_scratch`)
	})
	if _, err := db.ExecContext(ctx, `TRUNCATE transactor_scratch`); err != nil {
		t.Fatalf("truncating scratch table: %v", err)
	}

	transactor := NewBunTransactor(db)

	t.Run("commits on success", func(t *testing.T) {
		err := transactor.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			_, err := tx.ExecContext(ctx, `INSERT INTO transactor_scratch (id) VALUES (1)`)
			return err
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM transactor_scratch WHERE id = 1`).Scan(&count); err != nil {
			t.Fatalf("querying scratch table: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected row to be committed, got count %d", count)
		}
	})

	t.Run("rolls back on error", func(t *testing.T) {
		wantErr := errors.New("boom")
		err := transactor.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `INSERT INTO transactor_scratch (id) VALUES (2)`); err != nil {
				return err
			}
			return wantErr
		})
		if err == nil {
			t.Fatal("expected an error, got nil")
		}

		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM transactor_scratch WHERE id = 2`).Scan(&count); err != nil {
			t.Fatalf("querying scratch table: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected row to be rolled back, got count %d", count)
		}
	})
}
