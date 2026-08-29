package postgres_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/uptrace/bun"

	"golden-app/backend/internal/database/dbtest"
	"golden-app/backend/users/domain"
	"golden-app/backend/users/internal/infrastructure/postgres"
)

func TestSessionIssuer_IssueAndValidate(t *testing.T) {
	db, _ := dbtest.New(t)
	issuer := postgres.NewSessionIssuer(db)
	ctx := context.Background()

	user := seedUser(t, db, "user-1", "ada@example.com")

	token, expiresAt, err := issuer.Issue(ctx, user.ID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty session token")
	}
	if !expiresAt.After(time.Now().UTC()) {
		t.Fatalf("expected a future expiry, got %s", expiresAt)
	}

	identity, _, err := issuer.Validate(ctx, token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if identity.UserID != user.ID || identity.Role != domain.RoleUser {
		t.Fatalf("unexpected identity: %+v", identity)
	}
}

// A user whose role has no role_permissions rows must resolve to an
// empty permission set, not an error and not a nil surprise: RBAC is
// opt-in, and most projects never seed the table.
func TestSessionIssuer_Validate_NoPermissionsSeeded(t *testing.T) {
	db, _ := dbtest.New(t)
	issuer := postgres.NewSessionIssuer(db)
	ctx := context.Background()

	user := seedUser(t, db, "user-1", "ada@example.com")
	token, _, err := issuer.Issue(ctx, user.ID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	identity, _, err := issuer.Validate(ctx, token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if identity.Permissions == nil {
		t.Fatal("expected an empty permission slice, got nil")
	}
	if len(identity.Permissions) != 0 {
		t.Fatalf("expected no permissions, got %v", identity.Permissions)
	}
	if identity.HasPermission("posts:delete") {
		t.Error("expected HasPermission to be false with no permissions seeded")
	}
}

func TestSessionIssuer_Validate_ResolvesRolePermissions(t *testing.T) {
	db, _ := dbtest.New(t)
	issuer := postgres.NewSessionIssuer(db)
	ctx := context.Background()

	user := seedUser(t, db, "user-1", "ada@example.com")
	seedPermission(t, db, "perm-1", "posts:delete", user.Role)
	seedPermission(t, db, "perm-2", "posts:edit", user.Role)
	// Granted to another role entirely — must not leak into this identity.
	seedPermission(t, db, "perm-3", "billing:refund", "admin")

	token, _, err := issuer.Issue(ctx, user.ID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	identity, _, err := issuer.Validate(ctx, token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(identity.Permissions) != 2 {
		t.Fatalf("expected 2 permissions, got %v", identity.Permissions)
	}
	if !identity.HasPermission("posts:delete") || !identity.HasPermission("posts:edit") {
		t.Fatalf("expected the role's permissions, got %v", identity.Permissions)
	}
	if identity.HasPermission("billing:refund") {
		t.Error("expected another role's permission not to leak in")
	}
}

func TestSessionIssuer_Validate_UnknownToken(t *testing.T) {
	db, _ := dbtest.New(t)
	issuer := postgres.NewSessionIssuer(db)

	_, _, err := issuer.Validate(context.Background(), "never-issued")
	if !errors.Is(err, domain.ErrSessionInvalid) {
		t.Fatalf("expected ErrSessionInvalid, got %v", err)
	}
}

func TestSessionIssuer_Validate_ExpiredSession(t *testing.T) {
	db, _ := dbtest.New(t)
	issuer := postgres.NewSessionIssuer(db)
	ctx := context.Background()

	user := seedUser(t, db, "user-1", "ada@example.com")
	token, _, err := issuer.Issue(ctx, user.ID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if _, err := db.NewRaw(
		`UPDATE sessions SET expires_at = ? WHERE id = ?`,
		time.Now().UTC().Add(-time.Minute), storedSessionID(token),
	).Exec(ctx); err != nil {
		t.Fatalf("expiring session: %v", err)
	}

	if _, _, err := issuer.Validate(ctx, token); !errors.Is(err, domain.ErrSessionInvalid) {
		t.Fatalf("expected ErrSessionInvalid, got %v", err)
	}
}

func TestSessionIssuer_Revoke(t *testing.T) {
	db, _ := dbtest.New(t)
	issuer := postgres.NewSessionIssuer(db)
	ctx := context.Background()

	user := seedUser(t, db, "user-1", "ada@example.com")
	token, _, err := issuer.Issue(ctx, user.ID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if err := issuer.Revoke(ctx, token); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, _, err := issuer.Validate(ctx, token); !errors.Is(err, domain.ErrSessionInvalid) {
		t.Fatalf("expected the revoked session to be invalid, got %v", err)
	}

	if err := issuer.Revoke(ctx, token); err != nil {
		t.Fatalf("expected revoking twice to be a no-op, got %v", err)
	}
}

func seedPermission(t *testing.T, db bun.IDB, id, key, role string) {
	t.Helper()
	ctx := context.Background()

	if _, err := db.NewRaw(`INSERT INTO permissions (id, key) VALUES (?, ?)`, id, key).Exec(ctx); err != nil {
		t.Fatalf("seeding permission: %v", err)
	}
	if _, err := db.NewRaw(
		`INSERT INTO role_permissions (role, permission_id) VALUES (?, ?)`, role, id,
	).Exec(ctx); err != nil {
		t.Fatalf("seeding role permission: %v", err)
	}
}

// storedSessionID is an independent restatement of the digest the
// adapter stores, so these tests verify the storage format rather
// than reusing the implementation's own helper.
func storedSessionID(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// The sessions table must never hold a directly usable credential: a
// leaked dump should yield only digests.
func TestSessionIssuer_StoresOnlyTheTokenDigest(t *testing.T) {
	db, _ := dbtest.New(t)
	issuer := postgres.NewSessionIssuer(db)
	ctx := context.Background()

	user := seedUser(t, db, "user-1", "ada@example.com")
	token, _, err := issuer.Issue(ctx, user.ID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	var ids []string
	if err := db.NewRaw(`SELECT id FROM sessions WHERE user_id = ?`, user.ID).Scan(ctx, &ids); err != nil {
		t.Fatalf("reading sessions: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected exactly 1 session row, got %d", len(ids))
	}
	if ids[0] == token {
		t.Fatal("expected the raw session token not to be stored")
	}
	if ids[0] != storedSessionID(token) {
		t.Fatalf("expected the stored id to be the token's SHA-256 digest, got %q", ids[0])
	}

	// The digest is what Validate looks up, so the raw token stays the
	// only thing that works — holding the stored value is not enough.
	if _, _, err := issuer.Validate(ctx, token); err != nil {
		t.Fatalf("expected the raw token to validate: %v", err)
	}
	if _, _, err := issuer.Validate(ctx, ids[0]); !errors.Is(err, domain.ErrSessionInvalid) {
		t.Fatal("expected the stored digest itself not to be usable as a token")
	}
}

// Validate resolves the account in the same query as the session, so
// GET /users/me and the login response need no follow-up lookup.
func TestSessionIssuer_Validate_ReturnsUser(t *testing.T) {
	db, _ := dbtest.New(t)
	issuer := postgres.NewSessionIssuer(db)
	repo := postgres.NewUserRepository(db)
	ctx := context.Background()

	seeded := seedUser(t, db, "user-1", "ada@example.com")
	verifiedAt := time.Now().UTC().Truncate(time.Microsecond)
	if err := repo.MarkEmailVerified(ctx, seeded.ID, verifiedAt); err != nil {
		t.Fatalf("mark email verified: %v", err)
	}

	token, _, err := issuer.Issue(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	identity, user, err := issuer.Validate(ctx, token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if user.ID != seeded.ID || user.Email != seeded.Email || user.Role != domain.RoleUser {
		t.Fatalf("unexpected user: %+v", user)
	}
	if !user.EmailVerified() {
		t.Fatal("expected the resolved user to report a verified email")
	}
	if !user.EmailVerifiedAt.Equal(verifiedAt) {
		t.Fatalf("expected EmailVerifiedAt %s, got %s", verifiedAt, user.EmailVerifiedAt)
	}
	if user.CreatedAt.IsZero() || user.UpdatedAt.IsZero() {
		t.Fatalf("expected timestamps to be populated, got %+v", user)
	}
	if identity.UserID != user.ID {
		t.Fatal("expected identity and user to agree on the id")
	}
}
