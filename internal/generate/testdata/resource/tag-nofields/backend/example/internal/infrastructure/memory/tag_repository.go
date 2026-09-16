package memory

import (
	"cmp"
	"context"
	"slices"
	"sync"

	"golden-app/backend/example/domain"
)

var _ domain.TagRepository = (*TagRepository)(nil)

// TagRepository is an in-memory, mutex-guarded domain.TagRepository.
type TagRepository struct {
	mu   sync.Mutex
	tags map[string]domain.Tag
}

// NewTagRepository constructs an empty in-memory TagRepository.
func NewTagRepository() *TagRepository {
	return &TagRepository{tags: make(map[string]domain.Tag)}
}

// Create stores tag, keyed by its ID.
func (r *TagRepository) Create(_ context.Context, tag domain.Tag) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tags[tag.ID] = tag
	return nil
}

// Get retrieves the Tag with the given id, or domain.ErrTagNotFound.
func (r *TagRepository) Get(_ context.Context, id string) (domain.Tag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tag, ok := r.tags[id]
	if !ok {
		return domain.Tag{}, domain.ErrTagNotFound
	}
	return tag, nil
}

// List returns up to limit Tags after skipping offset, newest first
// then by id, matching the Postgres adapter's order.
func (r *TagRepository) List(_ context.Context, limit, offset int) ([]domain.Tag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	all := make([]domain.Tag, 0, len(r.tags))
	for _, tag := range r.tags {
		all = append(all, tag)
	}
	slices.SortFunc(all, func(a, b domain.Tag) int {
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	if offset >= len(all) {
		return []domain.Tag{}, nil
	}
	all = all[offset:]
	if limit < len(all) {
		all = all[:limit]
	}
	return all, nil
}

// Update replaces the stored Tag's fields and UpdatedAt with tag's,
// keeping its CreatedAt, or returns domain.ErrTagNotFound.
func (r *TagRepository) Update(_ context.Context, tag domain.Tag) (domain.Tag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.tags[tag.ID]
	if !ok {
		return domain.Tag{}, domain.ErrTagNotFound
	}
	tag.CreatedAt = stored.CreatedAt
	r.tags[tag.ID] = tag
	return tag, nil
}

// Delete removes the Tag with the given id, or returns domain.ErrTagNotFound.
func (r *TagRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tags[id]; !ok {
		return domain.ErrTagNotFound
	}
	delete(r.tags, id)
	return nil
}
