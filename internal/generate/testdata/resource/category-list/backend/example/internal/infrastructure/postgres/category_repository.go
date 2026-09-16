package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"golden-app/backend/example/domain"
)

var _ domain.CategoryRepository = (*CategoryRepository)(nil)

// CategoryRepository is a Postgres-backed domain.CategoryRepository over a
// bun.IDB, so the same code runs against the pooled production *bun.DB
// and against a test transaction via dbtest.
type CategoryRepository struct {
	db bun.IDB
}

// NewCategoryRepository constructs a CategoryRepository backed by db.
func NewCategoryRepository(db bun.IDB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// categoryRow is Bun's row-mapping struct for the categories table. It
// stays in this package — domain.Category has no infrastructure imports
// or struct tags.
type categoryRow struct {
	bun.BaseModel `bun:"table:categories"`

	ID        string    `bun:"id,pk"`
	Name      string    `bun:"name"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}

func toCategoryRow(category domain.Category) categoryRow {
	return categoryRow{
		ID:        category.ID,
		Name:      category.Name,
		CreatedAt: category.CreatedAt,
		UpdatedAt: category.UpdatedAt,
	}
}

func fromCategoryRow(row categoryRow) domain.Category {
	return domain.Category{
		ID:        row.ID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

// Create stores category.
func (r *CategoryRepository) Create(ctx context.Context, category domain.Category) error {
	row := toCategoryRow(category)
	if _, err := r.db.NewInsert().Model(&row).Exec(ctx); err != nil {
		return fmt.Errorf("creating category: %w", err)
	}
	return nil
}

// List returns up to limit Categories after skipping offset, newest first then by id.
func (r *CategoryRepository) List(ctx context.Context, limit, offset int) ([]domain.Category, error) {
	rows := make([]categoryRow, 0)
	err := r.db.NewSelect().Model(&rows).Order("created_at DESC", "id").Limit(limit).Offset(offset).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing categories: %w", err)
	}
	categories := make([]domain.Category, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, fromCategoryRow(row))
	}
	return categories, nil
}
