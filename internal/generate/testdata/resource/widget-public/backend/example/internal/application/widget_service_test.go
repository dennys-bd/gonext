package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"golden-app/backend/example/domain"
	"golden-app/backend/example/internal/infrastructure/memory"
)

func TestWidgetService_Create(t *testing.T) {
	repo := memory.NewWidgetRepository()
	svc := NewWidgetService(repo)
	ctx := context.Background()

	created, err := svc.CreateWidget(ctx, "demo", 42, int64(42), 9.5, true, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	if err != nil {
		t.Fatalf("create: unexpected error: %v", err)
	}
	if created.ID == "" || created.Name != "demo" || created.Count != 42 || created.Total != int64(42) || created.Ratio != 9.5 || !created.Active || !created.Due.Equal(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Fatalf("unexpected widget: %+v", created)
	}
	if created.CreatedAt.IsZero() || !created.UpdatedAt.Equal(created.CreatedAt) {
		t.Fatalf("expected both timestamps set to the creation time, got %+v", created)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: unexpected error: %v", err)
	}
	if got != created {
		t.Fatalf("expected %+v, got %+v", created, got)
	}
}

func TestWidgetService_GetNotFound(t *testing.T) {
	svc := NewWidgetService(memory.NewWidgetRepository())

	_, err := svc.GetWidget(context.Background(), "does-not-exist")
	if !errors.Is(err, domain.ErrWidgetNotFound) {
		t.Fatalf("expected ErrWidgetNotFound, got %v", err)
	}
}
