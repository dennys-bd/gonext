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

var _ domain.WidgetRepository = (*WidgetRepository)(nil)

// WidgetRepository is a Postgres-backed domain.WidgetRepository over a
// bun.IDB, so the same code runs against the pooled production *bun.DB
// and against a test transaction via dbtest.
type WidgetRepository struct {
	db bun.IDB
}

// NewWidgetRepository constructs a WidgetRepository backed by db.
func NewWidgetRepository(db bun.IDB) *WidgetRepository {
	return &WidgetRepository{db: db}
}

// widgetRow is Bun's row-mapping struct for the widgets table. It
// stays in this package — domain.Widget has no infrastructure imports
// or struct tags.
type widgetRow struct {
	bun.BaseModel `bun:"table:widgets"`

	ID        string    `bun:"id,pk"`
	Name      string    `bun:"name"`
	Count     int       `bun:"count"`
	Total     int64     `bun:"total"`
	Ratio     float64   `bun:"ratio"`
	Active    bool      `bun:"active"`
	Due       time.Time `bun:"due"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}

func toWidgetRow(widget domain.Widget) widgetRow {
	return widgetRow{
		ID:        widget.ID,
		Name:      widget.Name,
		Count:     widget.Count,
		Total:     widget.Total,
		Ratio:     widget.Ratio,
		Active:    widget.Active,
		Due:       widget.Due,
		CreatedAt: widget.CreatedAt,
		UpdatedAt: widget.UpdatedAt,
	}
}

func fromWidgetRow(row widgetRow) domain.Widget {
	return domain.Widget{
		ID:        row.ID,
		Name:      row.Name,
		Count:     row.Count,
		Total:     row.Total,
		Ratio:     row.Ratio,
		Active:    row.Active,
		Due:       row.Due,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

// Create stores widget.
func (r *WidgetRepository) Create(ctx context.Context, widget domain.Widget) error {
	row := toWidgetRow(widget)
	if _, err := r.db.NewInsert().Model(&row).Exec(ctx); err != nil {
		return fmt.Errorf("creating widget: %w", err)
	}
	return nil
}

// Get retrieves the Widget with the given id, or domain.ErrWidgetNotFound.
func (r *WidgetRepository) Get(ctx context.Context, id string) (domain.Widget, error) {
	var row widgetRow
	if err := r.db.NewSelect().Model(&row).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Widget{}, domain.ErrWidgetNotFound
		}
		return domain.Widget{}, fmt.Errorf("querying widget: %w", err)
	}
	return fromWidgetRow(row), nil
}
