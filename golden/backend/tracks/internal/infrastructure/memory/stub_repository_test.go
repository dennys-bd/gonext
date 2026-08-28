package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"golden-app/backend/tracks/domain"
)

func TestStubRepository_CreateAndGet(t *testing.T) {
	repo := NewStubRepository()
	ctx := context.Background()
	s, err := domain.NewStub("id-1", "demo", time.Now().UTC())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("create: unexpected error: %v", err)
	}

	got, err := repo.Get(ctx, "id-1")
	if err != nil {
		t.Fatalf("get: unexpected error: %v", err)
	}
	if got != s {
		t.Fatalf("expected %+v, got %+v", s, got)
	}
}

func TestStubRepository_GetNotFound(t *testing.T) {
	repo := NewStubRepository()
	_, err := repo.Get(context.Background(), "does-not-exist")
	if !errors.Is(err, domain.ErrStubNotFound) {
		t.Fatalf("expected ErrStubNotFound, got %v", err)
	}
}
