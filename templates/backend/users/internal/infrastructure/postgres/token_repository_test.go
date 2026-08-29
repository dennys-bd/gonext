package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"[PROJECT-NAME]/backend/internal/database/dbtest"
	"[PROJECT-NAME]/backend/users/domain"
	"[PROJECT-NAME]/backend/users/internal/infrastructure/postgres"
)

func TestTokenRepository_CreateAndGet(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewTokenRepository(db)
	ctx := context.Background()

	user := seedUser(t, db, "user-1", "ada@example.com")
	now := time.Now().UTC().Truncate(time.Microsecond)
	token := domain.Token{
		ID:        "token-1",
		UserID:    user.ID,
		Kind:      domain.TokenKindEmailConfirmation,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}

	if err := repo.Create(ctx, token); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Get(ctx, token.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.UserID != token.UserID || got.Kind != token.Kind || !got.ExpiresAt.Equal(token.ExpiresAt) {
		t.Fatalf("expected %+v, got %+v", token, got)
	}
	if got.UsedAt != nil {
		t.Error("expected a new token to be unused")
	}
	if !got.Usable(domain.TokenKindEmailConfirmation, now) {
		t.Error("expected a fresh token to be usable")
	}
}

func TestTokenRepository_Get_NotFound(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewTokenRepository(db)

	_, err := repo.Get(context.Background(), "missing")
	if !errors.Is(err, domain.ErrTokenNotFound) {
		t.Fatalf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestTokenRepository_MarkUsed(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewTokenRepository(db)
	ctx := context.Background()

	user := seedUser(t, db, "user-1", "ada@example.com")
	now := time.Now().UTC().Truncate(time.Microsecond)
	token := domain.Token{
		ID:        "token-1",
		UserID:    user.ID,
		Kind:      domain.TokenKindPasswordReset,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}
	if err := repo.Create(ctx, token); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.MarkUsed(ctx, token.ID, now); err != nil {
		t.Fatalf("mark used: %v", err)
	}

	got, err := repo.Get(ctx, token.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.UsedAt == nil || !got.UsedAt.Equal(now) {
		t.Fatalf("expected UsedAt %s, got %v", now, got.UsedAt)
	}
	if got.Usable(domain.TokenKindPasswordReset, now) {
		t.Error("expected a consumed token to be unusable")
	}
}

// Consuming a token twice must fail, so two concurrent redemptions of
// the same token cannot both take effect.
func TestTokenRepository_MarkUsed_Twice(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewTokenRepository(db)
	ctx := context.Background()

	user := seedUser(t, db, "user-1", "ada@example.com")
	now := time.Now().UTC().Truncate(time.Microsecond)
	token := domain.Token{
		ID:        "token-1",
		UserID:    user.ID,
		Kind:      domain.TokenKindPasswordReset,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}
	if err := repo.Create(ctx, token); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.MarkUsed(ctx, token.ID, now); err != nil {
		t.Fatalf("first mark used: %v", err)
	}

	if err := repo.MarkUsed(ctx, token.ID, now); !errors.Is(err, domain.ErrTokenNotFound) {
		t.Fatalf("expected the second consumption to fail, got %v", err)
	}
}

func TestTokenRepository_MarkUsed_NotFound(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewTokenRepository(db)

	err := repo.MarkUsed(context.Background(), "missing", time.Now().UTC())
	if !errors.Is(err, domain.ErrTokenNotFound) {
		t.Fatalf("expected ErrTokenNotFound, got %v", err)
	}
}
