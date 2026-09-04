package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dennys-bd/gonext/auth"
	"github.com/uptrace/bun"

	"golden-app/backend/users/domain"
	"golden-app/backend/users/internal/idgen"
)

// SessionTTL is how long an issued session stays valid.
const SessionTTL = 30 * 24 * time.Hour

var _ domain.SessionIssuer = (*SessionIssuer)(nil)

// SessionIssuer is a Postgres-backed domain.SessionIssuer. Sessions
// are opaque random tokens; only their SHA-256 digest is stored, so a
// leaked database dump yields no usable session. Revoking deletes the
// row, so it takes effect immediately.
type SessionIssuer struct {
	db  bun.IDB
	ttl time.Duration
}

// NewSessionIssuer constructs a SessionIssuer backed by db, issuing
// sessions that live for SessionTTL.
func NewSessionIssuer(db bun.IDB) *SessionIssuer {
	return &SessionIssuer{db: db, ttl: SessionTTL}
}

type sessionRow struct {
	bun.BaseModel `bun:"table:sessions"`

	ID        string    `bun:"id,pk"`
	UserID    string    `bun:"user_id"`
	CreatedAt time.Time `bun:"created_at"`
	ExpiresAt time.Time `bun:"expires_at"`
}

// identityRow is one row of the Validate join: the session's user
// plus at most one permission granted to that user's role. A user
// whose role grants nothing still produces exactly one row, with a
// NULL permission key.
type identityRow struct {
	UserID          string         `bun:"user_id"`
	Email           string         `bun:"email"`
	Role            string         `bun:"role"`
	EmailVerifiedAt *time.Time     `bun:"email_verified_at"`
	CreatedAt       time.Time      `bun:"created_at"`
	UpdatedAt       time.Time      `bun:"updated_at"`
	PermissionKey   sql.NullString `bun:"permission_key"`
}

// Issue mints a random opaque session token for userID, returning the
// raw token to the caller while persisting only its digest.
func (i *SessionIssuer) Issue(ctx context.Context, userID string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(i.ttl)

	token, err := idgen.New()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generating session token: %w", err)
	}

	row := &sessionRow{ID: hashSessionToken(token), UserID: userID, CreatedAt: now, ExpiresAt: expiresAt}
	if _, err := i.db.NewInsert().Model(row).Exec(ctx); err != nil {
		return "", time.Time{}, fmt.Errorf("creating session: %w", err)
	}
	return token, expiresAt, nil
}

// Validate resolves token to an Identity and the account behind it,
// or domain.ErrSessionInvalid for an unknown or expired session. The
// user's record and the role's permissions come back from the same
// query, so neither a HasPermission check nor rendering the account
// costs another round-trip.
func (i *SessionIssuer) Validate(ctx context.Context, token string) (auth.Identity, domain.User, error) {
	var rows []identityRow

	err := i.db.NewRaw(`
		SELECT
			u.id                AS user_id,
			u.email             AS email,
			u.role              AS role,
			u.email_verified_at AS email_verified_at,
			u.created_at        AS created_at,
			u.updated_at        AS updated_at,
			p.key               AS permission_key
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		LEFT JOIN role_permissions rp ON rp.role = u.role
		LEFT JOIN permissions p ON p.id = rp.permission_id
		WHERE s.id = ? AND s.expires_at > ?
	`, hashSessionToken(token), time.Now().UTC()).Scan(ctx, &rows)
	if err != nil {
		return auth.Identity{}, domain.User{}, fmt.Errorf("querying session: %w", err)
	}
	if len(rows) == 0 {
		return auth.Identity{}, domain.User{}, domain.ErrSessionInvalid
	}

	permissions := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.PermissionKey.Valid {
			permissions = append(permissions, row.PermissionKey.String)
		}
	}

	// PasswordHash is deliberately not selected: nothing downstream of
	// a session lookup needs it, so it never leaves the users table.
	first := rows[0]
	identity := auth.Identity{UserID: first.UserID, Role: first.Role, Permissions: permissions}
	user := domain.User{
		ID:              first.UserID,
		Email:           first.Email,
		Role:            first.Role,
		EmailVerifiedAt: first.EmailVerifiedAt,
		CreatedAt:       first.CreatedAt,
		UpdatedAt:       first.UpdatedAt,
	}
	return identity, user, nil
}

// Revoke deletes the session identified by token. Revoking an unknown
// session is not an error, so logging out twice behaves the same as
// logging out once.
func (i *SessionIssuer) Revoke(ctx context.Context, token string) error {
	digest := hashSessionToken(token)
	if _, err := i.db.NewDelete().Model((*sessionRow)(nil)).Where("id = ?", digest).Exec(ctx); err != nil {
		return fmt.Errorf("revoking session: %w", err)
	}
	return nil
}

// hashSessionToken digests a raw session token for storage and
// lookup. SHA-256 rather than Argon2id on purpose: the input is 256
// random bits, so there is no dictionary to defend against and no
// reason to pay a slow hash on every request. What this buys is that
// a leaked sessions table contains no directly usable credential.
func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
