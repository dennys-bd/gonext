package domain

import (
	"context"
	"errors"
	"time"
)

// ErrProductNotFound is returned when a Product cannot be found by id.
var ErrProductNotFound = errors.New("example: product not found")

// Product is the example domain's product entity.
type Product struct {
	ID        string
	Title     string
	Price     int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProductRepository persists and retrieves Products.
type ProductRepository interface {
	Create(ctx context.Context, product Product) error
	Get(ctx context.Context, id string) (Product, error)
	List(ctx context.Context, limit, offset int) ([]Product, error)
	Update(ctx context.Context, product Product) (Product, error)
	Delete(ctx context.Context, id string) error
}
