package application

import (
	"context"
	"errors"
	"testing"

	"golden-app/backend/tracks/domain"
	"golden-app/backend/tracks/internal/infrastructure/memory"
)

func TestStubService_CreateAndGet(t *testing.T) {
	svc := NewStubService(memory.NewStubRepository())
	ctx := context.Background()

	created, err := svc.CreateStub(ctx, "demo")
	if err != nil {
		t.Fatalf("create: unexpected error: %v", err)
	}
	if created.ID == "" || created.Name != "demo" {
		t.Fatalf("unexpected stub: %+v", created)
	}

	got, err := svc.GetStub(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: unexpected error: %v", err)
	}
	if got != created {
		t.Fatalf("expected %+v, got %+v", created, got)
	}
}

func TestStubService_CreateEmptyName(t *testing.T) {
	svc := NewStubService(memory.NewStubRepository())
	_, err := svc.CreateStub(context.Background(), "")
	if !errors.Is(err, domain.ErrStubNameRequired) {
		t.Fatalf("expected ErrStubNameRequired, got %v", err)
	}
}

func TestStubService_GetNotFound(t *testing.T) {
	svc := NewStubService(memory.NewStubRepository())
	_, err := svc.GetStub(context.Background(), "does-not-exist")
	if !errors.Is(err, domain.ErrStubNotFound) {
		t.Fatalf("expected ErrStubNotFound, got %v", err)
	}
}
