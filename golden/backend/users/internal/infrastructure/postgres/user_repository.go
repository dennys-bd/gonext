package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/uptrace/bun"

	"golden-app/backend/users/domain"
)

var _ domain.UserRepository = (*UserRepository)(nil)

// UserRepository is a Postgres-backed domain.UserRepository.
type UserRepository struct {
	db bun.IDB
}

// NewUserRepository constructs a UserRepository backed by db.
func NewUserRepository(db bun.IDB) *UserRepository {
	return &UserRepository{db: db}
}

// userRow is Bun's row-mapping struct for the users table. It stays
// in this package — users/domain.User has no infrastructure imports
// or struct tags.
type userRow struct {
	bun.BaseModel `bun:"table:users"`

	ID              string     `bun:"id,pk"`
	Email           string     `bun:"email"`
	PasswordHash    string     `bun:"password_hash"`
	Role            string     `bun:"role"`
	EmailVerifiedAt *time.Time `bun:"email_verified_at"`
	CreatedAt       time.Time  `bun:"created_at"`
	UpdatedAt       time.Time  `bun:"updated_at"`
}

func (r userRow) toDomain() domain.User {
	return domain.User{
		ID:              r.ID,
		Email:           r.Email,
		PasswordHash:    r.PasswordHash,
		Role:            r.Role,
		EmailVerifiedAt: r.EmailVerifiedAt,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

// Create stores u.
func (r *UserRepository) Create(ctx context.Context, u domain.User) error {
	row := &userRow{
		ID:              u.ID,
		Email:           u.Email,
		PasswordHash:    u.PasswordHash,
		Role:            u.Role,
		EmailVerifiedAt: u.EmailVerifiedAt,
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
	}
	if _, err := r.db.NewInsert().Model(row).Exec(ctx); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEmailTaken
		}
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

// isUniqueViolation reports whether err is Postgres' unique_violation
// (SQLSTATE 23505). Letting the insert fail on the constraint is what
// makes registration race-proof: unlike a prior existence check, two
// concurrent inserts of the same email cannot both win.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// GetByID retrieves the User with the given id, or domain.ErrUserNotFound.
func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	return r.getBy(ctx, "id = ?", id)
}

// GetByEmail retrieves the User with the given email, or domain.ErrUserNotFound.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.getBy(ctx, "email = ?", domain.NormalizeEmail(email))
}

func (r *UserRepository) getBy(ctx context.Context, where string, arg any) (domain.User, error) {
	row := new(userRow)
	if err := r.db.NewSelect().Model(row).Where(where, arg).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("querying user: %w", err)
	}
	return row.toDomain(), nil
}

// UpdatePasswordHash replaces the stored password hash for id.
func (r *UserRepository) UpdatePasswordHash(ctx context.Context, id, passwordHash string, updatedAt time.Time) error {
	res, err := r.db.NewUpdate().
		Model((*userRow)(nil)).
		Set("password_hash = ?", passwordHash).
		Set("updated_at = ?", updatedAt).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("updating password hash: %w", err)
	}
	return requireOneRow(res, domain.ErrUserNotFound)
}

// MarkEmailVerified records verifiedAt as the confirmation time for id.
func (r *UserRepository) MarkEmailVerified(ctx context.Context, id string, verifiedAt time.Time) error {
	res, err := r.db.NewUpdate().
		Model((*userRow)(nil)).
		Set("email_verified_at = ?", verifiedAt).
		Set("updated_at = ?", verifiedAt).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("marking email verified: %w", err)
	}
	return requireOneRow(res, domain.ErrUserNotFound)
}

// requireOneRow turns an update that matched nothing into notFound,
// so a caller can tell "no such row" from "row left unchanged".
func requireOneRow(res sql.Result, notFound error) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("reading affected rows: %w", err)
	}
	if affected == 0 {
		return notFound
	}
	return nil
}
