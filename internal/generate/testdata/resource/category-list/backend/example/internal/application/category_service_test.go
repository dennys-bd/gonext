package application

import (
	"context"
	"testing"
	"time"

	"golden-app/backend/example/domain"
	"golden-app/backend/example/internal/infrastructure/memory"
)

func seedCategory(t *testing.T, repo *memory.CategoryRepository, id string) domain.Category {
	t.Helper()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	category := domain.Category{
		ID:        id,
		Name:      "demo",
		CreatedAt: at,
		UpdatedAt: at,
	}
	if err := repo.Create(context.Background(), category); err != nil {
		t.Fatalf("seeding a category: %v", err)
	}
	return category
}

func TestCategoryService_List(t *testing.T) {
	repo := memory.NewCategoryRepository()
	svc := NewCategoryService(repo)
	seeded := seedCategory(t, repo, "category-1")

	categories, err := svc.ListCategories(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("list: unexpected error: %v", err)
	}
	if len(categories) != 1 || categories[0] != seeded {
		t.Fatalf("expected [%+v], got %+v", seeded, categories)
	}
}
