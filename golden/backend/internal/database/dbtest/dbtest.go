// Package dbtest exercises Bun-backed repositories and services against a
// real Postgres database, rolling back every test's outer transaction so
// nothing is left behind.
package dbtest

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/uptrace/bun"

	"golden-app/backend/internal/database"
)

// DSN returns DATABASE_URL for a test that needs a real Postgres. It skips
// t unless ENV is test, so a bare `go test` in a dev shell never touches the
// dev database, and fails t if the database name does not end in _test.
func DSN(t *testing.T) string {
	t.Helper()

	if os.Getenv("ENV") != "test" {
		t.Skip("ENV is not test; run `make test` (it selects mise.test.toml) to run this test")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; run `make test` to run this test")
	}
	// A DATABASE_URL override in mise.local.toml wins over mise.test.toml
	// too, so ENV alone cannot prove this is not someone's dev database.
	if u, err := url.Parse(dsn); err != nil || !strings.HasSuffix(u.Path, "_test") {
		t.Fatalf("DATABASE_URL must name a database ending in _test under ENV=test, got %q", dsn)
	}
	return dsn
}

// New opens an outer transaction against DSN(t), returned as a bun.IDB
// plus a Transactor built on it via SAVEPOINTs; both are rolled back
// automatically via t.Cleanup.
func New(t *testing.T) (bun.IDB, database.Transactor) {
	t.Helper()

	db, err := database.Connect(context.Background(), DSN(t))
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
