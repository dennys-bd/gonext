package domain

import (
	"context"
	"errors"
	"time"
)

// ErrTagNotFound is returned when a Tag cannot be found by id.
var ErrTagNotFound = errors.New("example: tag not found")

// Tag is the example domain's tag entity.
type Tag struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TagRepository persists and retrieves Tags.
type TagRepository interface {
	Create(ctx context.Context, tag Tag) error
	Get(ctx context.Context, id string) (Tag, error)
	List(ctx context.Context, limit, offset int) ([]Tag, error)
	Update(ctx context.Context, tag Tag) (Tag, error)
	Delete(ctx context.Context, id string) error
}
