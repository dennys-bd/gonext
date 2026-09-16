package postgres_test

import (
	"context"
	"testing"
	"time"

	"golden-app/backend/example/domain"
	"golden-app/backend/example/internal/infrastructure/postgres"
	"golden-app/backend/internal/database/dbtest"
)

func testCategory(id string, at time.Time) domain.Category {
	return domain.Category{
		ID:        id,
		Name:      "demo",
		CreatedAt: at,
		UpdatedAt: at,
	}
}

func sameCategory(a, b domain.Category) bool {
	return a.ID == b.ID &&
		a.Name == b.Name &&
		a.CreatedAt.Equal(b.CreatedAt) &&
		a.UpdatedAt.Equal(b.UpdatedAt)
}

func TestCategoryRepository_Create(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewCategoryRepository(db)

	if err := repo.Create(context.Background(), testCategory("category-1", time.Now().UTC().Truncate(time.Microsecond))); err != nil {
		t.Fatalf("create: %v", err)
	}
}

func TestCategoryRepository_List(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewCategoryRepository(db)
	ctx := context.Background()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	older := testCategory("category-a", base)
	newer := testCategory("category-c", base.Add(time.Hour))
	newerLowerID := testCategory("category-b", base.Add(time.Hour))
	for _, category := range []domain.Category{older, newer, newerLowerID} {
		if err := repo.Create(ctx, category); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	all, err := repo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 3 || !sameCategory(all[0], newerLowerID) || !sameCategory(all[1], newer) || !sameCategory(all[2], older) {
		t.Fatalf("expected newest first then by id, got %+v", all)
	}

	page, err := repo.List(ctx, 1, 1)
	if err != nil {
		t.Fatalf("list page: %v", err)
	}
	if len(page) != 1 || !sameCategory(page[0], newer) {
		t.Fatalf("expected [%+v], got %+v", newer, page)
	}

	past, err := repo.List(ctx, 10, 5)
	if err != nil {
		t.Fatalf("list past the end: %v", err)
	}
	if past == nil || len(past) != 0 {
		t.Fatalf("expected an empty, non-nil slice, got %+v", past)
	}
}
