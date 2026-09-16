package memory

import (
	"context"
	"testing"
	"time"

	"golden-app/backend/example/domain"
)

func testCategory(id string, at time.Time) domain.Category {
	return domain.Category{
		ID:        id,
		Name:      "demo",
		CreatedAt: at,
		UpdatedAt: at,
	}
}

func TestCategoryRepository_Create(t *testing.T) {
	repo := NewCategoryRepository()

	if err := repo.Create(context.Background(), testCategory("category-1", time.Now().UTC())); err != nil {
		t.Fatalf("create: unexpected error: %v", err)
	}
}

func TestCategoryRepository_List(t *testing.T) {
	repo := NewCategoryRepository()
	ctx := context.Background()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	older := testCategory("category-a", base)
	newer := testCategory("category-c", base.Add(time.Hour))
	newerLowerID := testCategory("category-b", base.Add(time.Hour))
	for _, category := range []domain.Category{older, newer, newerLowerID} {
		if err := repo.Create(ctx, category); err != nil {
			t.Fatalf("create: unexpected error: %v", err)
		}
	}

	all, err := repo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list: unexpected error: %v", err)
	}
	if len(all) != 3 || all[0] != newerLowerID || all[1] != newer || all[2] != older {
		t.Fatalf("expected newest first then by id, got %+v", all)
	}

	page, err := repo.List(ctx, 1, 1)
	if err != nil {
		t.Fatalf("list page: unexpected error: %v", err)
	}
	if len(page) != 1 || page[0] != newer {
		t.Fatalf("expected [%+v], got %+v", newer, page)
	}

	past, err := repo.List(ctx, 10, 5)
	if err != nil {
		t.Fatalf("list past the end: unexpected error: %v", err)
	}
	if past == nil || len(past) != 0 {
		t.Fatalf("expected an empty, non-nil slice, got %+v", past)
	}
}
