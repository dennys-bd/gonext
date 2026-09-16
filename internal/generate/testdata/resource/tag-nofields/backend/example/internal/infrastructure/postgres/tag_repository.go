package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"golden-app/backend/example/domain"
)

var _ domain.TagRepository = (*TagRepository)(nil)

// TagRepository is a Postgres-backed domain.TagRepository over a
// bun.IDB, so the same code runs against the pooled production *bun.DB
// and against a test transaction via dbtest.
type TagRepository struct {
	db bun.IDB
}

// NewTagRepository constructs a TagRepository backed by db.
func NewTagRepository(db bun.IDB) *TagRepository {
	return &TagRepository{db: db}
}

// tagRow is Bun's row-mapping struct for the tags table. It
// stays in this package — domain.Tag has no infrastructure imports
// or struct tags.
type tagRow struct {
	bun.BaseModel `bun:"table:tags"`

	ID        string    `bun:"id,pk"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}

func toTagRow(tag domain.Tag) tagRow {
	return tagRow{
		ID:        tag.ID,
		CreatedAt: tag.CreatedAt,
		UpdatedAt: tag.UpdatedAt,
	}
}

func fromTagRow(row tagRow) domain.Tag {
	return domain.Tag{
		ID:        row.ID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

// Create stores tag.
func (r *TagRepository) Create(ctx context.Context, tag domain.Tag) error {
	row := toTagRow(tag)
	if _, err := r.db.NewInsert().Model(&row).Exec(ctx); err != nil {
		return fmt.Errorf("creating tag: %w", err)
	}
	return nil
}

// Get retrieves the Tag with the given id, or domain.ErrTagNotFound.
func (r *TagRepository) Get(ctx context.Context, id string) (domain.Tag, error) {
	var row tagRow
	if err := r.db.NewSelect().Model(&row).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Tag{}, domain.ErrTagNotFound
		}
		return domain.Tag{}, fmt.Errorf("querying tag: %w", err)
	}
	return fromTagRow(row), nil
}

// List returns up to limit Tags after skipping offset, newest first then by id.
func (r *TagRepository) List(ctx context.Context, limit, offset int) ([]domain.Tag, error) {
	rows := make([]tagRow, 0)
	err := r.db.NewSelect().Model(&rows).Order("created_at DESC", "id").Limit(limit).Offset(offset).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}
	tags := make([]domain.Tag, 0, len(rows))
	for _, row := range rows {
		tags = append(tags, fromTagRow(row))
	}
	return tags, nil
}

// Update replaces the stored Tag's fields and UpdatedAt with tag's
// in one write and returns the stored row, or domain.ErrTagNotFound.
func (r *TagRepository) Update(ctx context.Context, tag domain.Tag) (domain.Tag, error) {
	row := toTagRow(tag)
	err := r.db.NewUpdate().Model(&row).Column("updated_at").WherePK().Returning("*").Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Tag{}, domain.ErrTagNotFound
		}
		return domain.Tag{}, fmt.Errorf("updating tag: %w", err)
	}
	return fromTagRow(row), nil
}

// Delete removes the Tag with the given id, or returns domain.ErrTagNotFound.
func (r *TagRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.NewDelete().Model(&tagRow{ID: id}).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("deleting tag: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleting tag: %w", err)
	}
	if affected == 0 {
		return domain.ErrTagNotFound
	}
	return nil
}
