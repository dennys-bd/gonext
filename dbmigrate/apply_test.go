package dbmigrate

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// openTestDB opens a bun.DB against TEST_DATABASE_URL, which must
// point at a throwaway database: the helper drops bun_migrations and
// bun_migration_locks both before returning and again on cleanup, so
// pointing it at a real project's database would destroy its
// migration bookkeeping.
func openTestDB(t *testing.T) *bun.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	sqldb, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	db := bun.NewDB(sqldb, pgdialect.New())

	reset := func() {
		if _, err := db.ExecContext(context.Background(), "DROP TABLE IF EXISTS bun_migrations, bun_migration_locks"); err != nil {
			t.Fatalf("resetting migration tables: %v", err)
		}
	}
	reset()
	t.Cleanup(func() {
		reset()
		db.Close()
	})
	return db
}

func readNames(t *testing.T, db *bun.DB) []string {
	t.Helper()
	var names []string
	if err := db.NewSelect().ColumnExpr("name").Table("bun_migrations").OrderExpr("id").Scan(context.Background(), &names); err != nil {
		t.Fatalf("reading applied names: %v", err)
	}
	return names
}

func TestApply_AppliesInOrderAndRecordsNames(t *testing.T) {
	db := openTestDB(t)

	var ran []string
	up := func(key string) MigrationFunc {
		return func(ctx context.Context, db *bun.DB) error {
			ran = append(ran, key)
			return nil
		}
	}
	down := func(ctx context.Context, db *bun.DB) error { return nil }

	r := NewRegistry()
	mustAdd(t, r, Migration{Domain: "a", Version: "0001", Name: "x"}, up("a/0001"), down)
	mustAdd(t, r, Migration{Domain: "a", Version: "0002", Name: "y"}, up("a/0002"), down)
	mustAdd(t, r, Migration{Domain: "b", Version: "0001", Name: "z"}, up("b/0001"), down, Dependency{Domain: "a", Version: "0002"})

	applied, err := r.Apply(context.Background(), db)
	if err != nil {
		t.Fatalf("Apply: unexpected error: %v", err)
	}

	wantKeys := []string{"a/0001", "a/0002", "b/0001"}
	gotKeys := make([]string, len(applied))
	for i, m := range applied {
		gotKeys[i] = m.String()
	}
	if !slices.Equal(gotKeys, wantKeys) {
		t.Errorf("applied = %v, want %v", gotKeys, wantKeys)
	}
	if !slices.Equal(ran, wantKeys) {
		t.Errorf("ran = %v, want %v", ran, wantKeys)
	}
	if got := readNames(t, db); !slices.Equal(got, wantKeys) {
		t.Errorf("readNames = %v, want %v", got, wantKeys)
	}
}

func TestApply_SecondCallAppliesNothing(t *testing.T) {
	db := openTestDB(t)

	var ran []string
	up := func(key string) MigrationFunc {
		return func(ctx context.Context, db *bun.DB) error {
			ran = append(ran, key)
			return nil
		}
	}
	down := func(ctx context.Context, db *bun.DB) error { return nil }

	r := NewRegistry()
	mustAdd(t, r, Migration{Domain: "a", Version: "0001", Name: "x"}, up("a/0001"), down)
	mustAdd(t, r, Migration{Domain: "a", Version: "0002", Name: "y"}, up("a/0002"), down)

	if _, err := r.Apply(context.Background(), db); err != nil {
		t.Fatalf("first Apply: unexpected error: %v", err)
	}
	ranAfterFirst := append([]string(nil), ran...)

	applied, err := r.Apply(context.Background(), db)
	if err != nil {
		t.Fatalf("second Apply: unexpected error: %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("second Apply: applied = %v, want none", applied)
	}
	if !slices.Equal(ran, ranAfterFirst) {
		t.Errorf("second Apply: ran changed to %v, want unchanged %v", ran, ranAfterFirst)
	}
}

func TestApply_StopsAtFailingUpAndLeavesItUnmarked(t *testing.T) {
	db := openTestDB(t)

	errBoom := errors.New("boom")
	down := func(ctx context.Context, db *bun.DB) error { return nil }

	r := NewRegistry()
	mustAdd(t, r, Migration{Domain: "a", Version: "0001", Name: "x"}, func(ctx context.Context, db *bun.DB) error { return nil }, down)
	mustAdd(t, r, Migration{Domain: "a", Version: "0002", Name: "create_x"}, func(ctx context.Context, db *bun.DB) error { return errBoom }, down)
	mustAdd(t, r, Migration{Domain: "a", Version: "0003", Name: "z"}, func(ctx context.Context, db *bun.DB) error { return nil }, down)

	applied, err := r.Apply(context.Background(), db)
	if !errors.Is(err, errBoom) {
		t.Fatalf("Apply: err = %v, want wrapping %v", err, errBoom)
	}
	wantPrefix := "applying a/0002 create_x: "
	if got := err.Error(); !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("Apply: err = %q, want prefix %q", got, wantPrefix)
	}

	wantApplied := []string{"a/0001"}
	gotApplied := make([]string, len(applied))
	for i, m := range applied {
		gotApplied[i] = m.String()
	}
	if !slices.Equal(gotApplied, wantApplied) {
		t.Errorf("applied = %v, want %v", gotApplied, wantApplied)
	}
	if got := readNames(t, db); !slices.Equal(got, wantApplied) {
		t.Errorf("readNames = %v, want %v", got, wantApplied)
	}

	var lockCount int
	if err := db.NewSelect().ColumnExpr("count(*)").Table("bun_migration_locks").Scan(context.Background(), &lockCount); err != nil {
		t.Fatalf("counting locks: %v", err)
	}
	if lockCount != 0 {
		t.Errorf("bun_migration_locks count = %d, want 0", lockCount)
	}
}

func mustAdd(t *testing.T, r *Registry, m Migration, up, down MigrationFunc, deps ...Dependency) {
	t.Helper()
	if err := r.Add(m, up, down, deps...); err != nil {
		t.Fatalf("Add(%s): unexpected error: %v", m, err)
	}
}

// migrateTestRegistry builds users/0001, users/0002, users/0003,
// orders/0001 After(users,0002), orders/0002 — the same shape
// TestRegistry_Plan exercises — with up/down that record their key
// (prefixed "up:"/"down:") to calls, in call order.
func migrateTestRegistry(t *testing.T, calls *[]string) *Registry {
	t.Helper()
	up := func(key string) MigrationFunc {
		return func(ctx context.Context, db *bun.DB) error {
			*calls = append(*calls, "up:"+key)
			return nil
		}
	}
	down := func(key string) MigrationFunc {
		return func(ctx context.Context, db *bun.DB) error {
			*calls = append(*calls, "down:"+key)
			return nil
		}
	}

	r := NewRegistry()
	mustAdd(t, r, Migration{Domain: "users", Version: "0001", Name: "x"}, up("users/0001"), down("users/0001"))
	mustAdd(t, r, Migration{Domain: "users", Version: "0002", Name: "x"}, up("users/0002"), down("users/0002"))
	mustAdd(t, r, Migration{Domain: "users", Version: "0003", Name: "x"}, up("users/0003"), down("users/0003"))
	mustAdd(t, r, Migration{Domain: "orders", Version: "0001", Name: "x"}, up("orders/0001"), down("orders/0001"), Dependency{Domain: "users", Version: "0002"})
	mustAdd(t, r, Migration{Domain: "orders", Version: "0002", Name: "x"}, up("orders/0002"), down("orders/0002"))
	return r
}

func migrationStrings(ms []Migration) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.String()
	}
	return out
}

func TestMigrate_ForwardToTargetAppliesOnlyClosure(t *testing.T) {
	db := openTestDB(t)
	var calls []string
	r := migrateTestRegistry(t, &calls)

	rolledBack, applied, err := r.Migrate(context.Background(), db, Target{Domain: "orders", Version: "0002"}, nil)
	if err != nil {
		t.Fatalf("Migrate: unexpected error: %v", err)
	}

	if len(rolledBack) != 0 {
		t.Errorf("rolledBack = %v, want none", rolledBack)
	}
	wantApplied := []string{"users/0001", "users/0002", "orders/0001", "orders/0002"}
	if got := migrationStrings(applied); !slices.Equal(got, wantApplied) {
		t.Errorf("applied = %v, want %v", got, wantApplied)
	}
	wantCalls := []string{"up:users/0001", "up:users/0002", "up:orders/0001", "up:orders/0002"}
	if !slices.Equal(calls, wantCalls) {
		t.Errorf("calls = %v, want %v", calls, wantCalls)
	}
	if got := readNames(t, db); !slices.Equal(got, wantApplied) {
		t.Errorf("readNames = %v, want %v", got, wantApplied)
	}
}

func TestMigrate_BackwardRollsBackDependentsFirstAndUnmarks(t *testing.T) {
	db := openTestDB(t)
	var calls []string
	r := migrateTestRegistry(t, &calls)

	if _, err := r.Apply(context.Background(), db); err != nil {
		t.Fatalf("Apply: unexpected error: %v", err)
	}
	calls = nil // only care about Migrate's own calls below

	var confirmedWith []Migration
	confirm := func(rollback []Migration) bool {
		confirmedWith = rollback
		return true
	}

	rolledBack, applied, err := r.Migrate(context.Background(), db, Target{Domain: "users", Version: "0001"}, confirm)
	if err != nil {
		t.Fatalf("Migrate: unexpected error: %v", err)
	}

	wantRolledBack := []string{"orders/0002", "orders/0001", "users/0003", "users/0002"}
	if got := migrationStrings(rolledBack); !slices.Equal(got, wantRolledBack) {
		t.Errorf("rolledBack = %v, want %v", got, wantRolledBack)
	}
	if got := migrationStrings(confirmedWith); !slices.Equal(got, wantRolledBack) {
		t.Errorf("confirm was called with %v, want %v", got, wantRolledBack)
	}
	if len(applied) != 0 {
		t.Errorf("applied = %v, want none", applied)
	}

	wantCalls := []string{"down:orders/0002", "down:orders/0001", "down:users/0003", "down:users/0002"}
	if !slices.Equal(calls, wantCalls) {
		t.Errorf("calls = %v, want %v", calls, wantCalls)
	}
	if got := readNames(t, db); !slices.Equal(got, []string{"users/0001"}) {
		t.Errorf("readNames = %v, want [users/0001]", got)
	}
}

// TestMigrate_StateChangedBetweenPlanAndLockAborts uses the confirm
// callback — which runs between planning and locking — to stand in
// for a concurrent run that changes the applied set. The confirmed
// plan no longer matches, so nothing must execute.
func TestMigrate_StateChangedBetweenPlanAndLockAborts(t *testing.T) {
	db := openTestDB(t)
	var calls []string
	r := migrateTestRegistry(t, &calls)

	// Only users/0001..0002 applied, so a rollback to users/0001 is
	// planned as [users/0002].
	if _, _, err := r.Migrate(context.Background(), db, Target{Domain: "users", Version: "0002"}, nil); err != nil {
		t.Fatalf("Migrate (setup): unexpected error: %v", err)
	}
	calls = nil

	confirm := func(rollback []Migration) bool {
		// The "concurrent run": apply everything else before the lock.
		if _, err := r.Apply(context.Background(), db); err != nil {
			t.Fatalf("Apply (concurrent): unexpected error: %v", err)
		}
		calls = nil
		return true
	}

	rolledBack, applied, err := r.Migrate(context.Background(), db, Target{Domain: "users", Version: "0001"}, confirm)
	if !errors.Is(err, ErrStateChanged) {
		t.Fatalf("Migrate: err = %v, want %v", err, ErrStateChanged)
	}
	if rolledBack != nil || applied != nil {
		t.Errorf("Migrate: rolledBack = %v, applied = %v, want both nil", rolledBack, applied)
	}
	if len(calls) != 0 {
		t.Errorf("Migrate: up/down calls after confirm = %v, want none", calls)
	}
	want := []string{"users/0001", "users/0002", "orders/0001", "orders/0002", "users/0003"}
	if got := readNames(t, db); !slices.Equal(got, want) {
		t.Errorf("readNames = %v, want the concurrent run's set %v", got, want)
	}
}

func TestMigrate_NotConfirmedTouchesNothing(t *testing.T) {
	db := openTestDB(t)
	var calls []string
	r := migrateTestRegistry(t, &calls)

	if _, err := r.Apply(context.Background(), db); err != nil {
		t.Fatalf("Apply: unexpected error: %v", err)
	}
	beforeNames := readNames(t, db)
	calls = nil

	confirm := func(rollback []Migration) bool { return false }

	rolledBack, applied, err := r.Migrate(context.Background(), db, Target{Domain: "users", Version: "0001"}, confirm)
	if !errors.Is(err, ErrNotConfirmed) {
		t.Fatalf("Migrate: err = %v, want %v", err, ErrNotConfirmed)
	}
	if rolledBack != nil || applied != nil {
		t.Errorf("Migrate: rolledBack = %v, applied = %v, want both nil", rolledBack, applied)
	}
	if len(calls) != 0 {
		t.Errorf("Migrate: up/down calls = %v, want none", calls)
	}
	if got := readNames(t, db); !slices.Equal(got, beforeNames) {
		t.Errorf("readNames changed to %v, want unchanged %v", got, beforeNames)
	}
}
