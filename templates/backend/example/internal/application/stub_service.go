// Package application holds the example domain's use cases. Depends
// only on example/domain — never imports infrastructure or presentation.
package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"[PROJECT-NAME]/backend/example/domain"
)

// StubService implements the example domain's stub use cases.
type StubService struct {
	repo domain.StubRepository
}

// NewStubService constructs a StubService backed by repo.
func NewStubService(repo domain.StubRepository) *StubService {
	return &StubService{repo: repo}
}

// CreateStub creates and persists a new Stub with the given name.
func (s *StubService) CreateStub(ctx context.Context, name string) (domain.Stub, error) {
	stub, err := domain.NewStub(newStubID(), name, time.Now().UTC())
	if err != nil {
		return domain.Stub{}, err
	}
	if err := s.repo.Create(ctx, stub); err != nil {
		return domain.Stub{}, err
	}
	return stub, nil
}

// GetStub retrieves a Stub by id.
func (s *StubService) GetStub(ctx context.Context, id string) (domain.Stub, error) {
	return s.repo.Get(ctx, id)
}

// newStubID generates a random hex id. Stdlib-only (no uuid
// dependency) since this is a placeholder example, not a real ID
// strategy.
func newStubID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
