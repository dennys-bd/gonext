package domain

import (
	"context"
	"errors"
)

// ErrStubNotFound is returned when a Stub cannot be found by id.
var ErrStubNotFound = errors.New("tracks: stub not found")

// StubRepository persists and retrieves Stubs.
type StubRepository interface {
	Create(ctx context.Context, s Stub) error
	Get(ctx context.Context, id string) (Stub, error)
}
