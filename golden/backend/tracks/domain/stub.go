// Package domain holds the tracks domain's public entities, value
// objects, domain errors, and repository ports. It depends on nothing
// outside the standard library.
package domain

import (
	"errors"
	"time"
)

// ErrStubNameRequired is returned when constructing a Stub with an empty name.
var ErrStubNameRequired = errors.New("tracks: stub name is required")

// Stub is the tracks domain's placeholder entity.
type Stub struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

// NewStub constructs a Stub, validating that name is non-empty.
func NewStub(id, name string, createdAt time.Time) (Stub, error) {
	if name == "" {
		return Stub{}, ErrStubNameRequired
	}
	return Stub{ID: id, Name: name, CreatedAt: createdAt}, nil
}
