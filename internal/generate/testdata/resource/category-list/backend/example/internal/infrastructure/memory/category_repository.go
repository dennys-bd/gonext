package memory

import (
	"cmp"
	"context"
	"slices"
	"sync"

	"golden-app/backend/example/domain"
)

var _ domain.CategoryRepository = (*CategoryRepository)(nil)

// CategoryRepository is an in-memory, mutex-guarded domain.CategoryRepository.
type CategoryRepository struct {
	mu         sync.Mutex
	categories map[string]domain.Category
}

// NewCategoryRepository constructs an empty in-memory CategoryRepository.
func NewCategoryRepository() *CategoryRepository {
	return &CategoryRepository{categories: make(map[string]domain.Category)}
}

// Create stores category, keyed by its ID.
func (r *CategoryRepository) Create(_ context.Context, category domain.Category) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.categories[category.ID] = category
	return nil
}

// List returns up to limit Categories after skipping offset, newest first
// then by id, matching the Postgres adapter's order.
func (r *CategoryRepository) List(_ context.Context, limit, offset int) ([]domain.Category, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	all := make([]domain.Category, 0, len(r.categories))
	for _, category := range r.categories {
		all = append(all, category)
	}
	slices.SortFunc(all, func(a, b domain.Category) int {
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	if offset >= len(all) {
		return []domain.Category{}, nil
	}
	all = all[offset:]
	if limit < len(all) {
		all = all[:limit]
	}
	return all, nil
}
