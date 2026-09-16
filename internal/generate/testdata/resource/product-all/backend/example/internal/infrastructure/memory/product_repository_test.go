package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"golden-app/backend/example/domain"
)

func testProduct(id string, at time.Time) domain.Product {
	return domain.Product{
		ID:        id,
		Title:     "demo",
		Price:     42,
		CreatedAt: at,
		UpdatedAt: at,
	}
}

func TestProductRepository_CreateAndGet(t *testing.T) {
	repo := NewProductRepository()
	ctx := context.Background()
	product := testProduct("product-1", time.Now().UTC())

	if err := repo.Create(ctx, product); err != nil {
		t.Fatalf("create: unexpected error: %v", err)
	}

	got, err := repo.Get(ctx, product.ID)
	if err != nil {
		t.Fatalf("get: unexpected error: %v", err)
	}
	if got != product {
		t.Fatalf("expected %+v, got %+v", product, got)
	}
}

func TestProductRepository_GetNotFound(t *testing.T) {
	repo := NewProductRepository()

	_, err := repo.Get(context.Background(), "does-not-exist")
	if !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

func TestProductRepository_List(t *testing.T) {
	repo := NewProductRepository()
	ctx := context.Background()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	older := testProduct("product-a", base)
	newer := testProduct("product-c", base.Add(time.Hour))
	newerLowerID := testProduct("product-b", base.Add(time.Hour))
	for _, product := range []domain.Product{older, newer, newerLowerID} {
		if err := repo.Create(ctx, product); err != nil {
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

func TestProductRepository_Update(t *testing.T) {
	repo := NewProductRepository()
	ctx := context.Background()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	product := testProduct("product-1", at)
	if err := repo.Create(ctx, product); err != nil {
		t.Fatalf("create: unexpected error: %v", err)
	}

	updated, err := repo.Update(ctx, domain.Product{
		ID:        product.ID,
		Title:     "demo-updated",
		Price:     43,
		UpdatedAt: at.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("update: unexpected error: %v", err)
	}
	if updated.Title != "demo-updated" || updated.Price != 43 {
		t.Fatalf("expected the fields replaced, got %+v", updated)
	}
	if !updated.CreatedAt.Equal(at) || !updated.UpdatedAt.Equal(at.Add(time.Hour)) {
		t.Fatalf("expected CreatedAt kept and UpdatedAt replaced, got %+v", updated)
	}
}

func TestProductRepository_UpdateNotFound(t *testing.T) {
	repo := NewProductRepository()

	_, err := repo.Update(context.Background(), testProduct("does-not-exist", time.Now().UTC()))
	if !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

func TestProductRepository_Delete(t *testing.T) {
	repo := NewProductRepository()
	ctx := context.Background()
	product := testProduct("product-1", time.Now().UTC())
	if err := repo.Create(ctx, product); err != nil {
		t.Fatalf("create: unexpected error: %v", err)
	}

	if err := repo.Delete(ctx, product.ID); err != nil {
		t.Fatalf("delete: unexpected error: %v", err)
	}
	if err := repo.Delete(ctx, product.ID); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound after delete, got %v", err)
	}
}
