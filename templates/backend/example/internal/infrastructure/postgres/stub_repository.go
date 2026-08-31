// Package postgres provides the example domain's Postgres-backed
// adapter for its repository ports.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"[PROJECT-NAME]/backend/example/domain"
)

var _ domain.StubRepository = (*StubRepository)(nil)

// StubRepository is a Postgres-backed domain.StubRepository, built
// over a bun.IDB so the same code runs against the pooled production
// *bun.DB and against a test transaction via dbtest.
type StubRepository struct {
	db bun.IDB
}

// NewStubRepository constructs a StubRepository backed by db.
func NewStubRepository(db bun.IDB) *StubRepository {
	return &StubRepository{db: db}
}

// stubRow is Bun's row-mapping struct for the stubs table. It stays
// in this package — example/domain.Stub has no infrastructure imports
// or struct tags.
type stubRow struct {
	bun.BaseModel `bun:"table:stubs"`

	ID        string    `bun:"id,pk"`
	Name      string    `bun:"name"`
	CreatedAt time.Time `bun:"created_at"`
}

// Create stores s.
func (r *StubRepository) Create(ctx context.Context, s domain.Stub) error {
	row := &stubRow{ID: s.ID, Name: s.Name, CreatedAt: s.CreatedAt}
	if _, err := r.db.NewInsert().Model(row).Exec(ctx); err != nil {
		return fmt.Errorf("creating stub: %w", err)
	}
	return nil
}

// Get retrieves the Stub with the given id, or domain.ErrStubNotFound.
func (r *StubRepository) Get(ctx context.Context, id string) (domain.Stub, error) {
	row := new(stubRow)
	if err := r.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Stub{}, domain.ErrStubNotFound
		}
		return domain.Stub{}, fmt.Errorf("querying stub: %w", err)
	}
	return domain.Stub{ID: row.ID, Name: row.Name, CreatedAt: row.CreatedAt}, nil
}
