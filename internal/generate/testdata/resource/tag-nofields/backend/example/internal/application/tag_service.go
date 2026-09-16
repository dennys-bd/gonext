package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"golden-app/backend/example/domain"
)

// TagService implements the example domain's tag use cases.
type TagService struct {
	repo domain.TagRepository
}

// NewTagService constructs a TagService backed by repo.
func NewTagService(repo domain.TagRepository) *TagService {
	return &TagService{repo: repo}
}

// CreateTag creates and persists a new Tag.
func (s *TagService) CreateTag(ctx context.Context) (domain.Tag, error) {
	now := time.Now().UTC()
	tag := domain.Tag{
		ID:        newTagID(),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, tag); err != nil {
		return domain.Tag{}, err
	}
	return tag, nil
}

// GetTag retrieves a Tag by id.
func (s *TagService) GetTag(ctx context.Context, id string) (domain.Tag, error) {
	return s.repo.Get(ctx, id)
}

// ListTags retrieves up to limit Tags after skipping offset, newest first.
func (s *TagService) ListTags(ctx context.Context, limit, offset int) ([]domain.Tag, error) {
	return s.repo.List(ctx, limit, offset)
}

// UpdateTag replaces the fields of the Tag with the given id and returns the stored result.
func (s *TagService) UpdateTag(ctx context.Context, id string) (domain.Tag, error) {
	return s.repo.Update(ctx, domain.Tag{
		ID:        id,
		UpdatedAt: time.Now().UTC(),
	})
}

// DeleteTag removes the Tag with the given id.
func (s *TagService) DeleteTag(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// newTagID generates a random hex id.
func newTagID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
