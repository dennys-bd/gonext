package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"golden-app/backend/example/domain"
)

// ProductService implements the example domain's product use cases.
type ProductService struct {
	repo domain.ProductRepository
}

// NewProductService constructs a ProductService backed by repo.
func NewProductService(repo domain.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

// CreateProduct creates and persists a new Product.
func (s *ProductService) CreateProduct(ctx context.Context, title string, price int) (domain.Product, error) {
	now := time.Now().UTC()
	product := domain.Product{
		ID:        newProductID(),
		Title:     title,
		Price:     price,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, product); err != nil {
		return domain.Product{}, err
	}
	return product, nil
}

// GetProduct retrieves a Product by id.
func (s *ProductService) GetProduct(ctx context.Context, id string) (domain.Product, error) {
	return s.repo.Get(ctx, id)
}

// ListProducts retrieves up to limit Products after skipping offset, newest first.
func (s *ProductService) ListProducts(ctx context.Context, limit, offset int) ([]domain.Product, error) {
	return s.repo.List(ctx, limit, offset)
}

// UpdateProduct replaces the fields of the Product with the given id and returns the stored result.
func (s *ProductService) UpdateProduct(ctx context.Context, id string, title string, price int) (domain.Product, error) {
	return s.repo.Update(ctx, domain.Product{
		ID:        id,
		Title:     title,
		Price:     price,
		UpdatedAt: time.Now().UTC(),
	})
}

// DeleteProduct removes the Product with the given id.
func (s *ProductService) DeleteProduct(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// newProductID generates a random hex id.
func newProductID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
