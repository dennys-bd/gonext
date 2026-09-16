package memory

import (
	"cmp"
	"context"
	"slices"
	"sync"

	"golden-app/backend/example/domain"
)

var _ domain.ProductRepository = (*ProductRepository)(nil)

// ProductRepository is an in-memory, mutex-guarded domain.ProductRepository.
type ProductRepository struct {
	mu       sync.Mutex
	products map[string]domain.Product
}

// NewProductRepository constructs an empty in-memory ProductRepository.
func NewProductRepository() *ProductRepository {
	return &ProductRepository{products: make(map[string]domain.Product)}
}

// Create stores product, keyed by its ID.
func (r *ProductRepository) Create(_ context.Context, product domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.products[product.ID] = product
	return nil
}

// Get retrieves the Product with the given id, or domain.ErrProductNotFound.
func (r *ProductRepository) Get(_ context.Context, id string) (domain.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	product, ok := r.products[id]
	if !ok {
		return domain.Product{}, domain.ErrProductNotFound
	}
	return product, nil
}

// List returns up to limit Products after skipping offset, newest first
// then by id, matching the Postgres adapter's order.
func (r *ProductRepository) List(_ context.Context, limit, offset int) ([]domain.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	all := make([]domain.Product, 0, len(r.products))
	for _, product := range r.products {
		all = append(all, product)
	}
	slices.SortFunc(all, func(a, b domain.Product) int {
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	if offset >= len(all) {
		return []domain.Product{}, nil
	}
	all = all[offset:]
	if limit < len(all) {
		all = all[:limit]
	}
	return all, nil
}

// Update replaces the stored Product's fields and UpdatedAt with product's,
// keeping its CreatedAt, or returns domain.ErrProductNotFound.
func (r *ProductRepository) Update(_ context.Context, product domain.Product) (domain.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.products[product.ID]
	if !ok {
		return domain.Product{}, domain.ErrProductNotFound
	}
	product.CreatedAt = stored.CreatedAt
	r.products[product.ID] = product
	return product, nil
}

// Delete removes the Product with the given id, or returns domain.ErrProductNotFound.
func (r *ProductRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.products[id]; !ok {
		return domain.ErrProductNotFound
	}
	delete(r.products, id)
	return nil
}
