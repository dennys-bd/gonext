package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golden-app/backend/internal/database/dbtest"
	"golden-app/backend/example/domain"
	"golden-app/backend/example/internal/infrastructure/postgres"
)

func TestStubRepository_CreateAndGet(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewStubRepository(db)

	stub, err := domain.NewStub("stub-1", "demo", time.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		t.Fatalf("constructing stub: %v", err)
	}

	if err := repo.Create(context.Background(), stub); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Get(context.Background(), stub.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != stub.ID || got.Name != stub.Name || !got.CreatedAt.Equal(stub.CreatedAt) {
		t.Fatalf("expected %+v, got %+v", stub, got)
	}
}

func TestStubRepository_Get_NotFound(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewStubRepository(db)

	_, err := repo.Get(context.Background(), "missing")
	if !errors.Is(err, domain.ErrStubNotFound) {
		t.Fatalf("expected ErrStubNotFound, got %v", err)
	}
}
