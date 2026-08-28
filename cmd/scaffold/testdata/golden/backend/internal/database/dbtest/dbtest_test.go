package dbtest_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/uptrace/bun"

	"golden-app/backend/internal/database"
	"golden-app/backend/internal/database/dbtest"
)

func requireTestDB(t *testing.T) {
	t.Helper()
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set; run `make db-up` and export it to run this test")
	}
}

func setupScratchTable(t *testing.T) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	db, err := database.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connecting to set up scratch table: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("closing setup database connection: %v", err)
		}
	}()

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS dbtest_scratch (id int PRIMARY KEY)`); err != nil {
		t.Fatalf("creating scratch table: %v", err)
	}
	if _, err := db.ExecContext(ctx, `TRUNCATE dbtest_scratch`); err != nil {
		t.Fatalf("truncating scratch table: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DROP TABLE IF EXISTS dbtest_scratch`)
	})
}

func countScratchRows(t *testing.T) int {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	db, err := database.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connecting to count scratch rows: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("closing count database connection: %v", err)
		}
	}()

	var count int
	if err := db.QueryRowContext(context.Background(), `SELECT count(*) FROM dbtest_scratch`).Scan(&count); err != nil {
		t.Fatalf("counting scratch rows: %v", err)
	}
	return count
}

func countScratchRowsIn(t *testing.T, tx bun.IDB) int {
	t.Helper()
	var count int
	if err := tx.QueryRowContext(context.Background(), `SELECT count(*) FROM dbtest_scratch`).Scan(&count); err != nil {
		t.Fatalf("counting scratch rows in tx: %v", err)
	}
	return count
}

func TestNew_RollsBackAfterCleanup(t *testing.T) {
	requireTestDB(t)
	setupScratchTable(t)

	t.Run("insert inside the harness transaction", func(t *testing.T) {
		tx, _ := dbtest.New(t)
		if _, err := tx.ExecContext(context.Background(), `INSERT INTO dbtest_scratch (id) VALUES (1)`); err != nil {
			t.Fatalf("insert: %v", err)
		}
		if got := countScratchRowsIn(t, tx); got != 1 {
			t.Fatalf("expected 1 row visible inside the transaction, got %d", got)
		}
	})

	// t.Run above only returns once the subtest's own t.Cleanup (dbtest's
	// rollback) has already run.
	if got := countScratchRows(t); got != 0 {
		t.Fatalf("expected 0 rows after cleanup rolled back the transaction, got %d", got)
	}
}

func TestNew_TransactorCommitsAndRollsBackWithinOuterTx(t *testing.T) {
	requireTestDB(t)
	setupScratchTable(t)

	t.Run("commit and rollback via the savepoint transactor", func(t *testing.T) {
		tx, transactor := dbtest.New(t)

		if err := transactor.RunInTx(context.Background(), nil, func(ctx context.Context, sptx bun.Tx) error {
			_, err := sptx.ExecContext(ctx, `INSERT INTO dbtest_scratch (id) VALUES (2)`)
			return err
		}); err != nil {
			t.Fatalf("committing via transactor: %v", err)
		}

		if err := transactor.RunInTx(context.Background(), nil, func(ctx context.Context, sptx bun.Tx) error {
			if _, err := sptx.ExecContext(ctx, `INSERT INTO dbtest_scratch (id) VALUES (3)`); err != nil {
				return err
			}
			return errors.New("boom")
		}); err == nil {
			t.Fatal("expected an error from the failing savepoint transaction")
		}

		if got := countScratchRowsIn(t, tx); got != 1 {
			t.Fatalf("expected only the committed row (id=2) visible, got count %d", got)
		}
	})

	if got := countScratchRows(t); got != 0 {
		t.Fatalf("expected 0 rows after the outer transaction rolled back, got %d", got)
	}
}
