// Package dbtest provides a test harness for exercising Bun-backed
// repositories and services against a real Postgres database without
// leaving any data behind: every test runs inside one outer
// transaction that the harness rolls back in t.Cleanup.
package dbtest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/uptrace/bun"

	"personal-life/backend/internal/database"
)

// New opens an outer transaction against TEST_DATABASE_URL and
// returns it as a bun.IDB for a repository under test, plus a
// Transactor built on the same transaction via SAVEPOINTs. The
// transaction is rolled back automatically via t.Cleanup, so tests
// need no manual data cleanup.
//
// New skips t if TEST_DATABASE_URL is not set.
func New(t *testing.T) (bun.IDB, database.Transactor) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; run `make db-up` and `make migrate-up` against it to run this test")
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

// savepointTransactor is database.Transactor implemented via
// SAVEPOINTs on an existing *bun.Tx, since Postgres does not support
// a real nested BEGIN on one connection. It lets a service under test
// call Transactor.RunInTx and observe the same commit/rollback
// semantics it would get from a real *bun.DB, while everything still
// nests inside dbtest's outer transaction that New rolls back.
type savepointTransactor struct {
	tx  bun.Tx
	seq int
}

// RunInTx runs fn inside a SAVEPOINT on the outer transaction,
// releasing it on success or rolling back to it on error. name is
// built from a package-local counter, never from external input, so
// string-building the SAVEPOINT/RELEASE statements is safe.
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
