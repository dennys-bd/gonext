package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golden-app/backend/example/domain"
	"golden-app/backend/example/internal/infrastructure/postgres"
	"golden-app/backend/internal/database/dbtest"
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

func sameWidget(a, b domain.Widget) bool {
	return a.ID == b.ID &&
		a.Name == b.Name &&
		a.Count == b.Count &&
		a.Total == b.Total &&
		a.Ratio == b.Ratio &&
		a.Active == b.Active &&
		a.Due.Equal(b.Due) &&
		a.CreatedAt.Equal(b.CreatedAt) &&
		a.UpdatedAt.Equal(b.UpdatedAt)
}

func TestWidgetRepository_CreateAndGet(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewWidgetRepository(db)
	ctx := context.Background()
	widget := testWidget("widget-1", time.Now().UTC().Truncate(time.Microsecond))

	if err := repo.Create(ctx, widget); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Get(ctx, widget.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !sameWidget(got, widget) {
		t.Fatalf("expected %+v, got %+v", widget, got)
	}
}

func TestWidgetRepository_GetNotFound(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewWidgetRepository(db)

	_, err := repo.Get(context.Background(), "does-not-exist")
	if !errors.Is(err, domain.ErrWidgetNotFound) {
		t.Fatalf("expected ErrWidgetNotFound, got %v", err)
	}
}
