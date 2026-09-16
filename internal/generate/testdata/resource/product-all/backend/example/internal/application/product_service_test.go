package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"golden-app/backend/example/domain"
	"golden-app/backend/example/internal/infrastructure/memory"
)

func seedProduct(t *testing.T, repo *memory.ProductRepository, id string) domain.Product {
	t.Helper()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	product := domain.Product{
		ID:        id,
		Title:     "demo",
		Price:     42,
		CreatedAt: at,
		UpdatedAt: at,
	}
	if err := repo.Create(context.Background(), product); err != nil {
		t.Fatalf("seeding a product: %v", err)
	}
	return product
}

func TestProductService_Create(t *testing.T) {
	repo := memory.NewProductRepository()
	svc := NewProductService(repo)
	ctx := context.Background()

	created, err := svc.CreateProduct(ctx, "demo", 42)
	if err != nil {
		t.Fatalf("create: unexpected error: %v", err)
	}
	if created.ID == "" || created.Title != "demo" || created.Price != 42 {
		t.Fatalf("unexpected product: %+v", created)
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

func TestProductService_GetNotFound(t *testing.T) {
	svc := NewProductService(memory.NewProductRepository())

	_, err := svc.GetProduct(context.Background(), "does-not-exist")
	if !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

func TestProductService_List(t *testing.T) {
	repo := memory.NewProductRepository()
	svc := NewProductService(repo)
	seeded := seedProduct(t, repo, "product-1")

	products, err := svc.ListProducts(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("list: unexpected error: %v", err)
	}
	if len(products) != 1 || products[0] != seeded {
		t.Fatalf("expected [%+v], got %+v", seeded, products)
	}
}

func TestProductService_Update(t *testing.T) {
	repo := memory.NewProductRepository()
	svc := NewProductService(repo)
	seeded := seedProduct(t, repo, "product-1")

	updated, err := svc.UpdateProduct(context.Background(), seeded.ID, "demo-updated", 43)
	if err != nil {
		t.Fatalf("update: unexpected error: %v", err)
	}
	if updated.ID != seeded.ID || updated.Title != "demo-updated" || updated.Price != 43 {
		t.Fatalf("unexpected product: %+v", updated)
	}
	if !updated.CreatedAt.Equal(seeded.CreatedAt) || !updated.UpdatedAt.After(seeded.UpdatedAt) {
		t.Fatalf("expected CreatedAt kept and UpdatedAt advanced, got %+v", updated)
	}
}

func TestProductService_UpdateNotFound(t *testing.T) {
	svc := NewProductService(memory.NewProductRepository())

	_, err := svc.UpdateProduct(context.Background(), "does-not-exist", "demo-updated", 43)
	if !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

func TestProductService_Delete(t *testing.T) {
	repo := memory.NewProductRepository()
	svc := NewProductService(repo)
	seeded := seedProduct(t, repo, "product-1")
	ctx := context.Background()

	if err := svc.DeleteProduct(ctx, seeded.ID); err != nil {
		t.Fatalf("delete: unexpected error: %v", err)
	}
	if err := svc.DeleteProduct(ctx, seeded.ID); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound after delete, got %v", err)
	}
}
