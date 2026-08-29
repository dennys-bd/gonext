package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golden-app/backend/internal/database/dbtest"
	"golden-app/backend/users/domain"
	"golden-app/backend/users/internal/infrastructure/postgres"
)

func TestTxRunner_CommitsBothWrites(t *testing.T) {
	db, transactor := dbtest.New(t)
	runner := postgres.NewTxRunner(transactor)
	store := postgres.NewStore(db)
	ctx := context.Background()

	user := newUser(t, "user-1", "ada@example.com")
	now := time.Now().UTC().Truncate(time.Microsecond)
	token := domain.Token{
		ID:        "token-1",
		UserID:    user.ID,
		Kind:      domain.TokenKindEmailConfirmation,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}

	err := runner.RunInTx(ctx, func(ctx context.Context, s domain.Store) error {
		if err := s.Users().Create(ctx, user); err != nil {
			return err
		}
		return s.Tokens().Create(ctx, token)
	})
	if err != nil {
		t.Fatalf("run in tx: %v", err)
	}

	if _, err := store.Users().GetByID(ctx, user.ID); err != nil {
		t.Fatalf("expected the user to be committed: %v", err)
	}
	if _, err := store.Tokens().Get(ctx, token.ID); err != nil {
		t.Fatalf("expected the token to be committed: %v", err)
	}
}

// Registration writes a user and a token together; a failure partway
// through must leave neither behind.
func TestTxRunner_RollsBackOnError(t *testing.T) {
	db, transactor := dbtest.New(t)
	runner := postgres.NewTxRunner(transactor)
	store := postgres.NewStore(db)
	ctx := context.Background()

	user := newUser(t, "user-1", "ada@example.com")
	wantErr := errors.New("second write failed")

	err := runner.RunInTx(ctx, func(ctx context.Context, s domain.Store) error {
		if err := s.Users().Create(ctx, user); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected the fn's error to surface, got %v", err)
	}

	if _, err := store.Users().GetByID(ctx, user.ID); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("expected the user write to be rolled back, got %v", err)
	}
}
