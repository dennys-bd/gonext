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

var _ domain.ProductRepository = (*ProductRepository)(nil)

// ProductRepository is a Postgres-backed domain.ProductRepository over a
// bun.IDB, so the same code runs against the pooled production *bun.DB
// and against a test transaction via dbtest.
type ProductRepository struct {
	db bun.IDB
}

// NewProductRepository constructs a ProductRepository backed by db.
func NewProductRepository(db bun.IDB) *ProductRepository {
	return &ProductRepository{db: db}
}

// productRow is Bun's row-mapping struct for the products table. It
// stays in this package — domain.Product has no infrastructure imports
// or struct tags.
type productRow struct {
	bun.BaseModel `bun:"table:products"`

	ID        string    `bun:"id,pk"`
	Title     string    `bun:"title"`
	Price     int       `bun:"price"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}

func toProductRow(product domain.Product) productRow {
	return productRow{
		ID:        product.ID,
		Title:     product.Title,
		Price:     product.Price,
		CreatedAt: product.CreatedAt,
		UpdatedAt: product.UpdatedAt,
	}
}

func fromProductRow(row productRow) domain.Product {
	return domain.Product{
		ID:        row.ID,
		Title:     row.Title,
		Price:     row.Price,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

// Create stores product.
func (r *ProductRepository) Create(ctx context.Context, product domain.Product) error {
	row := toProductRow(product)
	if _, err := r.db.NewInsert().Model(&row).Exec(ctx); err != nil {
		return fmt.Errorf("creating product: %w", err)
	}
	return nil
}

// Get retrieves the Product with the given id, or domain.ErrProductNotFound.
func (r *ProductRepository) Get(ctx context.Context, id string) (domain.Product, error) {
	var row productRow
	if err := r.db.NewSelect().Model(&row).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Product{}, domain.ErrProductNotFound
		}
		return domain.Product{}, fmt.Errorf("querying product: %w", err)
	}
	return fromProductRow(row), nil
}

// List returns up to limit Products after skipping offset, newest first then by id.
func (r *ProductRepository) List(ctx context.Context, limit, offset int) ([]domain.Product, error) {
	rows := make([]productRow, 0)
	err := r.db.NewSelect().Model(&rows).Order("created_at DESC", "id").Limit(limit).Offset(offset).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing products: %w", err)
	}
	products := make([]domain.Product, 0, len(rows))
	for _, row := range rows {
		products = append(products, fromProductRow(row))
	}
	return products, nil
}

// Update replaces the stored Product's fields and UpdatedAt with product's
// in one write and returns the stored row, or domain.ErrProductNotFound.
func (r *ProductRepository) Update(ctx context.Context, product domain.Product) (domain.Product, error) {
	row := toProductRow(product)
	err := r.db.NewUpdate().Model(&row).Column("title", "price", "updated_at").WherePK().Returning("*").Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Product{}, domain.ErrProductNotFound
		}
		return domain.Product{}, fmt.Errorf("updating product: %w", err)
	}
	return fromProductRow(row), nil
}

// Delete removes the Product with the given id, or returns domain.ErrProductNotFound.
func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.NewDelete().Model(&productRow{ID: id}).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("deleting product: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleting product: %w", err)
	}
	if affected == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}
