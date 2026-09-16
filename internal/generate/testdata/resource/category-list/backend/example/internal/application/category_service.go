package application

import (
	"context"

	"golden-app/backend/example/domain"
)

// CategoryService implements the example domain's category use cases.
type CategoryService struct {
	repo domain.CategoryRepository
}

// NewCategoryService constructs a CategoryService backed by repo.
func NewCategoryService(repo domain.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

// ListCategories retrieves up to limit Categories after skipping offset, newest first.
func (s *CategoryService) ListCategories(ctx context.Context, limit, offset int) ([]domain.Category, error) {
	return s.repo.List(ctx, limit, offset)
}
