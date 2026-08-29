package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/uptrace/bun"

	"golden-app/backend/internal/database/dbtest"
	"golden-app/backend/users/domain"
	"golden-app/backend/users/internal/infrastructure/postgres"
)

const testPasswordHash = "$argon2id$v=19$m=65536,t=1,p=4$c2FsdHNhbHRzYWx0c2E$ZGlnZXN0"

func newUser(t *testing.T, id, email string) domain.User {
	t.Helper()

	u, err := domain.NewUser(id, email, testPasswordHash, domain.RoleUser, time.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		t.Fatalf("constructing user: %v", err)
	}
	return u
}

// seedUser inserts a user directly so tests that exercise other
// adapters have a valid foreign key to point at.
func seedUser(t *testing.T, db bun.IDB, id, email string) domain.User {
	t.Helper()

	user := newUser(t, id, email)
	if err := postgres.NewUserRepository(db).Create(context.Background(), user); err != nil {
		t.Fatalf("seeding user: %v", err)
	}
	return user
}

func TestUserRepository_CreateAndGet(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewUserRepository(db)
	ctx := context.Background()

	user := newUser(t, "user-1", "ada@example.com")
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("create: %v", err)
	}

	byID, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if byID.Email != user.Email || byID.Role != user.Role || byID.PasswordHash != user.PasswordHash {
		t.Fatalf("expected %+v, got %+v", user, byID)
	}
	if byID.EmailVerified() {
		t.Error("expected a new user to be unverified")
	}

	byEmail, err := repo.GetByEmail(ctx, "  ADA@Example.com ")
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if byEmail.ID != user.ID {
		t.Fatalf("expected lookup by email to normalize, got %+v", byEmail)
	}
}

// The email uniqueness decision belongs to the database constraint,
// not to a prior lookup — that is what makes registration race-proof.
func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewUserRepository(db)
	ctx := context.Background()

	if err := repo.Create(ctx, newUser(t, "user-1", "ada@example.com")); err != nil {
		t.Fatalf("first create: %v", err)
	}

	err := repo.Create(ctx, newUser(t, "user-2", "ada@example.com"))
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestUserRepository_NotFound(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewUserRepository(db)
	ctx := context.Background()

	if _, err := repo.GetByID(ctx, "missing"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("get by id: expected ErrUserNotFound, got %v", err)
	}
	if _, err := repo.GetByEmail(ctx, "missing@example.com"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("get by email: expected ErrUserNotFound, got %v", err)
	}
}

func TestUserRepository_UpdatePasswordHash(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewUserRepository(db)
	ctx := context.Background()

	user := seedUser(t, db, "user-1", "ada@example.com")
	updatedAt := time.Now().UTC().Truncate(time.Microsecond)

	if err := repo.UpdatePasswordHash(ctx, user.ID, "new-hash", updatedAt); err != nil {
		t.Fatalf("update password hash: %v", err)
	}

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.PasswordHash != "new-hash" {
		t.Fatalf("expected the new hash, got %q", got.PasswordHash)
	}
	if !got.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("expected UpdatedAt %s, got %s", updatedAt, got.UpdatedAt)
	}
}

func TestUserRepository_UpdatePasswordHash_NotFound(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewUserRepository(db)

	err := repo.UpdatePasswordHash(context.Background(), "missing", "new-hash", time.Now().UTC())
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserRepository_MarkEmailVerified(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewUserRepository(db)
	ctx := context.Background()

	user := seedUser(t, db, "user-1", "ada@example.com")
	verifiedAt := time.Now().UTC().Truncate(time.Microsecond)

	if err := repo.MarkEmailVerified(ctx, user.ID, verifiedAt); err != nil {
		t.Fatalf("mark email verified: %v", err)
	}

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if !got.EmailVerified() {
		t.Fatal("expected the email to be verified")
	}
	if !got.EmailVerifiedAt.Equal(verifiedAt) {
		t.Fatalf("expected EmailVerifiedAt %s, got %s", verifiedAt, got.EmailVerifiedAt)
	}
}

func TestUserRepository_MarkEmailVerified_NotFound(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewUserRepository(db)

	err := repo.MarkEmailVerified(context.Background(), "missing", time.Now().UTC())
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
