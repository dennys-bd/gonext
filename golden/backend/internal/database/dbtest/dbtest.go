// Package dbtest exercises Bun-backed repositories and services against a
// real Postgres database, rolling back every test's outer transaction so
// nothing is left behind.
package dbtest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/uptrace/bun"

	"golden-app/backend/internal/database"
)

// New opens an outer transaction against TEST_DATABASE_URL, returned as a
// bun.IDB plus a Transactor built on it via SAVEPOINTs; both are rolled
// back automatically via t.Cleanup. It skips t if TEST_DATABASE_URL is unset.
func New(t *testing.T) (bun.IDB, database.Transactor) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; run `make db-up` and `gonext migrate` against it to run this test")
	}

	db, err := database.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("closing test database connection: %v", err)
		}
	})

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("beginning outer test transaction: %v", err)
	}
	t.Cleanup(func() {
		if err := tx.Rollback(); err != nil {
			t.Errorf("rolling back outer test transaction: %v", err)
		}
	})

	return tx, &savepointTransactor{tx: tx}
}

// savepointTransactor implements database.Transactor via SAVEPOINTs on an
// existing *bun.Tx, since Postgres has no real nested BEGIN on one connection.
type savepointTransactor struct {
	tx  bun.Tx
	seq int
}

// RunInTx runs fn inside a SAVEPOINT, releasing it on success or rolling
// back to it on error. name comes from a package-local counter, never
// external input, so building it into SQL directly is safe.
func (s *savepointTransactor) RunInTx(ctx context.Context, _ *sql.TxOptions, fn func(context.Context, bun.Tx) error) error {
	s.seq++
	name := fmt.Sprintf("dbtest_sp_%d", s.seq)

	if _, err := s.tx.ExecContext(ctx, "SAVEPOINT "+name); err != nil {
		return fmt.Errorf("creating savepoint: %w", err)
	}

	if err := fn(ctx, s.tx); err != nil {
		if _, rbErr := s.tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT "+name); rbErr != nil {
			return fmt.Errorf("rolling back to savepoint after %w: %w", err, rbErr)
		}
		return err
	}

	if _, err := s.tx.ExecContext(ctx, "RELEASE SAVEPOINT "+name); err != nil {
		return fmt.Errorf("releasing savepoint: %w", err)
	}
	return nil
}
