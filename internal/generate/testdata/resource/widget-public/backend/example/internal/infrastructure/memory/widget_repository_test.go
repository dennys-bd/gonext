package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"golden-app/backend/example/domain"
)

func testWidget(id string, at time.Time) domain.Widget {
	return domain.Widget{
		ID:        id,
		Name:      "demo",
		Count:     42,
		Total:     int64(42),
		Ratio:     9.5,
		Active:    true,
		Due:       time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		CreatedAt: at,
		UpdatedAt: at,
	}
}

func TestWidgetRepository_CreateAndGet(t *testing.T) {
	repo := NewWidgetRepository()
	ctx := context.Background()
	widget := testWidget("widget-1", time.Now().UTC())

	if err := repo.Create(ctx, widget); err != nil {
		t.Fatalf("create: unexpected error: %v", err)
	}

	got, err := repo.Get(ctx, widget.ID)
	if err != nil {
		t.Fatalf("get: unexpected error: %v", err)
	}
	if got != widget {
		t.Fatalf("expected %+v, got %+v", widget, got)
	}
}

func TestWidgetRepository_GetNotFound(t *testing.T) {
	repo := NewWidgetRepository()

	_, err := repo.Get(context.Background(), "does-not-exist")
	if !errors.Is(err, domain.ErrWidgetNotFound) {
		t.Fatalf("expected ErrWidgetNotFound, got %v", err)
	}
}
