package domain

import (
	"context"
	"errors"
	"time"
)

// ErrCategoryNotFound is returned when a Category cannot be found by id.
var ErrCategoryNotFound = errors.New("example: category not found")

// Category is the example domain's category entity.
type Category struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CategoryRepository persists and retrieves Categories.
type CategoryRepository interface {
	List(ctx context.Context, limit, offset int) ([]Category, error)
}
