// Package memory provides an in-memory adapter for the example domain's
// repository ports.
package memory

import (
	"context"
	"sync"

	"[PROJECT-NAME]/backend/example/domain"
)

var _ domain.StubRepository = (*StubRepository)(nil)

// StubRepository is an in-memory, mutex-guarded domain.StubRepository.
type StubRepository struct {
	mu    sync.Mutex
	stubs map[string]domain.Stub
}

// NewStubRepository constructs an empty in-memory StubRepository.
func NewStubRepository() *StubRepository {
	return &StubRepository{stubs: make(map[string]domain.Stub)}
}

// Create stores s, keyed by its ID.
func (r *StubRepository) Create(_ context.Context, s domain.Stub) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stubs[s.ID] = s
	return nil
}

// Get retrieves the Stub with the given id, or domain.ErrStubNotFound.
func (r *StubRepository) Get(_ context.Context, id string) (domain.Stub, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.stubs[id]
	if !ok {
		return domain.Stub{}, domain.ErrStubNotFound
	}
	return s, nil
}
