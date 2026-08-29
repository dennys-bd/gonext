package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"[PROJECT-NAME]/backend/internal/database"
	"[PROJECT-NAME]/backend/users/domain"
	"[PROJECT-NAME]/backend/users/internal/application"
	"[PROJECT-NAME]/backend/users/internal/infrastructure/memory"
	"[PROJECT-NAME]/backend/users/internal/infrastructure/postgres"
)

// Registration's race-safety cannot be proven inside dbtest's single
// transaction — concurrent writers need their own connections — so
// this test opens a real pool and cleans up after itself.
func TestRegister_ConcurrentSameEmail_Postgres(t *testing.T) {
	const racers = 8

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; run `make db-up` and `make migrate-up` against it to run this test")
	}

	ctx := context.Background()
	db, err := database.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("closing test database connection: %v", err)
		}
	})

	// A unique address per run, so a previous run's leftovers cannot
	// decide this one's outcome.
	email := fmt.Sprintf("race-%d@example.com", time.Now().UnixNano())
	t.Cleanup(func() {
		if _, err := db.NewRaw(
			`DELETE FROM tokens WHERE user_id IN (SELECT id FROM users WHERE email = ?)`, email,
		).Exec(context.Background()); err != nil {
			t.Errorf("cleaning up tokens: %v", err)
		}
		if _, err := db.NewRaw(`DELETE FROM users WHERE email = ?`, email).Exec(context.Background()); err != nil {
			t.Errorf("cleaning up users: %v", err)
		}
	})

	svc := application.NewUserService(
		postgres.NewStore(db),
		postgres.NewTxRunner(database.NewBunTransactor(db)),
		postgres.NewSessionIssuer(db),
		memory.NewNotifier(),
		application.NewArgon2Hasher(),
		"test",
	)

	var (
		start   sync.WaitGroup
		done    sync.WaitGroup
		mu      sync.Mutex
		wins    int
		taken   int
		unknown []error
	)
	start.Add(1)
	done.Add(racers)

	for range racers {
		go func() {
			defer done.Done()
			start.Wait()

			_, err := svc.Register(ctx, email, "correct horse battery staple")

			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				wins++
			case errors.Is(err, domain.ErrEmailTaken):
				taken++
			default:
				unknown = append(unknown, err)
			}
		}()
	}

	start.Done()
	done.Wait()

	if len(unknown) != 0 {
		t.Fatalf("expected only nil or ErrEmailTaken, got %v", unknown)
	}
	if wins != 1 {
		t.Fatalf("expected exactly 1 winner, got %d (and %d rejected)", wins, taken)
	}
	if taken != racers-1 {
		t.Fatalf("expected %d rejections, got %d", racers-1, taken)
	}

	var count int
	if err := db.NewRaw(`SELECT count(*) FROM users WHERE email = ?`, email).Scan(ctx, &count); err != nil {
		t.Fatalf("counting users: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 account row, got %d", count)
	}

	// The loser transactions must roll back cleanly: no orphan
	// confirmation tokens from the rejected attempts.
	var tokens int
	if err := db.NewRaw(
		`SELECT count(*) FROM tokens WHERE user_id IN (SELECT id FROM users WHERE email = ?)`, email,
	).Scan(ctx, &tokens); err != nil {
		t.Fatalf("counting tokens: %v", err)
	}
	if tokens != 1 {
		t.Fatalf("expected exactly 1 confirmation token, got %d", tokens)
	}
}
