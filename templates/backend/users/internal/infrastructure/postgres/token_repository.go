package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"[PROJECT-NAME]/backend/users/domain"
)

var _ domain.TokenRepository = (*TokenRepository)(nil)

// TokenRepository is a Postgres-backed domain.TokenRepository.
type TokenRepository struct {
	db bun.IDB
}

// NewTokenRepository constructs a TokenRepository backed by db.
func NewTokenRepository(db bun.IDB) *TokenRepository {
	return &TokenRepository{db: db}
}

// tokenRow is Bun's row-mapping struct for the tokens table, which
// holds both the email confirmation and password reset flows,
// distinguished by kind.
type tokenRow struct {
	bun.BaseModel `bun:"table:tokens"`

	ID        string     `bun:"id,pk"`
	UserID    string     `bun:"user_id"`
	Kind      string     `bun:"kind"`
	CreatedAt time.Time  `bun:"created_at"`
	ExpiresAt time.Time  `bun:"expires_at"`
	UsedAt    *time.Time `bun:"used_at"`
}

// Create stores t.
func (r *TokenRepository) Create(ctx context.Context, t domain.Token) error {
	row := &tokenRow{
		ID:        t.ID,
		UserID:    t.UserID,
		Kind:      string(t.Kind),
		CreatedAt: t.CreatedAt,
		ExpiresAt: t.ExpiresAt,
		UsedAt:    t.UsedAt,
	}
	if _, err := r.db.NewInsert().Model(row).Exec(ctx); err != nil {
		return fmt.Errorf("creating token: %w", err)
	}
	return nil
}

// Get retrieves the Token with the given id, or domain.ErrTokenNotFound.
func (r *TokenRepository) Get(ctx context.Context, id string) (domain.Token, error) {
	row := new(tokenRow)
	if err := r.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Token{}, domain.ErrTokenNotFound
		}
		return domain.Token{}, fmt.Errorf("querying token: %w", err)
	}
	return domain.Token{
		ID:        row.ID,
		UserID:    row.UserID,
		Kind:      domain.TokenKind(row.Kind),
		CreatedAt: row.CreatedAt,
		ExpiresAt: row.ExpiresAt,
		UsedAt:    row.UsedAt,
	}, nil
}

// MarkUsed records usedAt as the consumption time for id. It only
// matches a token that has not already been consumed, so two
// concurrent redemptions cannot both succeed.
func (r *TokenRepository) MarkUsed(ctx context.Context, id string, usedAt time.Time) error {
	res, err := r.db.NewUpdate().
		Model((*tokenRow)(nil)).
		Set("used_at = ?", usedAt).
		Where("id = ?", id).
		Where("used_at IS NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("marking token used: %w", err)
	}
	return requireOneRow(res, domain.ErrTokenNotFound)
}
