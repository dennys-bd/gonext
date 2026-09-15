// Package application holds the users domain's use cases. Depends
// only on users/domain — never imports infrastructure or presentation.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dennys-bd/gonext/auth"

	"[PROJECT-NAME]/backend/users/domain"
	"[PROJECT-NAME]/backend/users/internal/idgen"
)

// One-shot token lifetimes.
const (
	confirmationTokenTTL = 24 * time.Hour
	resetTokenTTL        = time.Hour
)

// IsRelaxedEnv reports whether env relaxes safety for local development: raw
// tokens are returned, unconfirmed logins are allowed, and the cookie drops Secure.
// Everything else, including stg, is restricted; both the application and HTTP layers gate on this.
func IsRelaxedEnv(env string) bool {
	return env == "dev" || env == "test"
}

// UserService implements the users domain's account, session, and
// one-shot token use cases.
type UserService struct {
	store    domain.Store
	tx       domain.TxRunner
	issuer   domain.SessionIssuer
	notifier domain.Notifier
	hasher   domain.PasswordHasher
	env      string
}

// NewUserService constructs a UserService. env selects the behaviour
// boundary documented on IsRelaxedEnv.
func NewUserService(
	store domain.Store,
	tx domain.TxRunner,
	issuer domain.SessionIssuer,
	notifier domain.Notifier,
	hasher domain.PasswordHasher,
	env string,
) *UserService {
	return &UserService{store: store, tx: tx, issuer: issuer, notifier: notifier, hasher: hasher, env: env}
}

// RegisterResult is what Register returns. DevToken carries the raw
// confirmation token in a relaxed environment (see IsRelaxedEnv) and is empty otherwise.
type RegisterResult struct {
	DevToken string
}

// Register creates an account for email and issues an email confirmation token,
// or returns domain.ErrEmailTaken if the address is already registered.
// The uniqueness check happens at the database constraint, not a prior lookup, so
// concurrent registrations of the same address cannot both succeed.
func (s *UserService) Register(ctx context.Context, email, password string) (RegisterResult, error) {
	now := time.Now().UTC()

	email = domain.NormalizeEmail(email)
	if email == "" {
		return RegisterResult{}, domain.ErrEmailRequired
	}
	if err := domain.ValidatePassword(password); err != nil {
		return RegisterResult{}, err
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return RegisterResult{}, fmt.Errorf("hashing password: %w", err)
	}

	userID, err := idgen.New()
	if err != nil {
		return RegisterResult{}, fmt.Errorf("generating user id: %w", err)
	}
	user, err := domain.NewUser(userID, email, hash, domain.RoleUser, now)
	if err != nil {
		return RegisterResult{}, err
	}

	token, err := newToken(user.ID, domain.TokenKindEmailConfirmation, now, confirmationTokenTTL)
	if err != nil {
		return RegisterResult{}, err
	}

	err = s.tx.RunInTx(ctx, func(ctx context.Context, st domain.Store) error {
		if err := st.Users().Create(ctx, user); err != nil {
			if errors.Is(err, domain.ErrEmailTaken) {
				return err
			}
			return fmt.Errorf("creating user: %w", err)
		}
		if err := st.Tokens().Create(ctx, token); err != nil {
			return fmt.Errorf("creating confirmation token: %w", err)
		}
		return nil
	})
	if err != nil {
		return RegisterResult{}, err
	}

	if err := s.notifier.SendConfirmation(ctx, user.Email, token.ID); err != nil {
		return RegisterResult{}, fmt.Errorf("sending confirmation: %w", err)
	}
	return RegisterResult{DevToken: s.devToken(token.ID)}, nil
}

// Login verifies credentials and issues a session, returning the raw
// session token and its expiry.
func (s *UserService) Login(ctx context.Context, email, password string) (string, time.Time, error) {
	user, err := s.store.Users().GetByEmail(ctx, domain.NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Hash anyway so an unknown email costs the same as a known
			// one with a wrong password.
			_, _ = s.hasher.Hash(password)
			return "", time.Time{}, domain.ErrInvalidCredentials
		}
		return "", time.Time{}, fmt.Errorf("looking up user by email: %w", err)
	}

	ok, err := s.hasher.Verify(password, user.PasswordHash)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("verifying password: %w", err)
	}
	if !ok {
		return "", time.Time{}, domain.ErrInvalidCredentials
	}

	// Safe to reveal outside the generic error: the caller already
	// proved they know the password.
	if !IsRelaxedEnv(s.env) && !user.EmailVerified() {
		return "", time.Time{}, domain.ErrEmailNotConfirmed
	}

	token, expiresAt, err := s.issuer.Issue(ctx, user.ID)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("issuing session: %w", err)
	}
	return token, expiresAt, nil
}

// Logout revokes the session identified by token.
func (s *UserService) Logout(ctx context.Context, token string) error {
	if err := s.issuer.Revoke(ctx, token); err != nil {
		return fmt.Errorf("revoking session: %w", err)
	}
	return nil
}

// Me resolves a session token to its identity and the account behind
// it, in the single query SessionIssuer.Validate performs.
func (s *UserService) Me(ctx context.Context, token string) (auth.Identity, domain.User, error) {
	if token == "" {
		return auth.Identity{}, domain.User{}, domain.ErrSessionInvalid
	}

	identity, user, err := s.issuer.Validate(ctx, token)
	if err != nil {
		if errors.Is(err, domain.ErrSessionInvalid) {
			return auth.Identity{}, domain.User{}, err
		}
		return auth.Identity{}, domain.User{}, fmt.Errorf("validating session: %w", err)
	}
	return identity, user, nil
}

// Profile returns the account for userID.
func (s *UserService) Profile(ctx context.Context, userID string) (domain.User, error) {
	user, err := s.store.Users().GetByID(ctx, userID)
	if err != nil {
		return domain.User{}, fmt.Errorf("looking up user by id: %w", err)
	}
	return user, nil
}

// ConfirmEmail consumes an email confirmation token and marks the
// account's email verified.
func (s *UserService) ConfirmEmail(ctx context.Context, rawToken string) error {
	now := time.Now().UTC()

	token, err := s.consumableToken(ctx, rawToken, domain.TokenKindEmailConfirmation, now)
	if err != nil {
		return err
	}

	return s.tx.RunInTx(ctx, func(ctx context.Context, st domain.Store) error {
		if err := st.Tokens().MarkUsed(ctx, token.ID, now); err != nil {
			// A miss means a concurrent request already consumed the token —
			// same outcome as a stale token, not a server fault.
			if errors.Is(err, domain.ErrTokenNotFound) {
				return domain.ErrConfirmationTokenInvalid
			}
			return fmt.Errorf("consuming confirmation token: %w", err)
		}
		if err := st.Users().MarkEmailVerified(ctx, token.UserID, now); err != nil {
			return fmt.Errorf("marking email verified: %w", err)
		}
		return nil
	})
}

// RequestPasswordReset issues a password reset token for email. An unknown email
// is not an error and produces no token, so the caller cannot enumerate registered emails.
func (s *UserService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	now := time.Now().UTC()

	user, err := s.store.Users().GetByEmail(ctx, domain.NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("looking up user by email: %w", err)
	}

	token, err := newToken(user.ID, domain.TokenKindPasswordReset, now, resetTokenTTL)
	if err != nil {
		return "", err
	}
	if err := s.store.Tokens().Create(ctx, token); err != nil {
		return "", fmt.Errorf("creating reset token: %w", err)
	}

	// SendAccountExistsNotice is the port's reset-token message (email, resetToken);
	// reused here rather than widening the port.
	if err := s.notifier.SendAccountExistsNotice(ctx, user.Email, token.ID); err != nil {
		return "", fmt.Errorf("sending reset token: %w", err)
	}
	return s.devToken(token.ID), nil
}

// ConfirmPasswordReset consumes a password reset token and replaces
// the account's password.
func (s *UserService) ConfirmPasswordReset(ctx context.Context, rawToken, newPassword string) error {
	now := time.Now().UTC()

	if err := domain.ValidatePassword(newPassword); err != nil {
		return err
	}

	token, err := s.consumableToken(ctx, rawToken, domain.TokenKindPasswordReset, now)
	if err != nil {
		return err
	}

	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	return s.tx.RunInTx(ctx, func(ctx context.Context, st domain.Store) error {
		if err := st.Tokens().MarkUsed(ctx, token.ID, now); err != nil {
			// See ConfirmEmail: a miss means a concurrent consumer won
			// the race, which is a stale token, not a server fault.
			if errors.Is(err, domain.ErrTokenNotFound) {
				return domain.ErrResetTokenInvalid
			}
			return fmt.Errorf("consuming reset token: %w", err)
		}
		if err := st.Users().UpdatePasswordHash(ctx, token.UserID, hash, now); err != nil {
			return fmt.Errorf("updating password: %w", err)
		}
		return nil
	})
}

// consumableToken collapses every failure into the single domain error for kind.
func (s *UserService) consumableToken(ctx context.Context, rawToken string, kind domain.TokenKind, now time.Time) (domain.Token, error) {
	invalid := invalidTokenError(kind)

	token, err := s.store.Tokens().Get(ctx, rawToken)
	if err != nil {
		if errors.Is(err, domain.ErrTokenNotFound) {
			return domain.Token{}, invalid
		}
		return domain.Token{}, fmt.Errorf("looking up token: %w", err)
	}
	if !token.Usable(kind, now) {
		return domain.Token{}, invalid
	}
	return token, nil
}

func invalidTokenError(kind domain.TokenKind) error {
	if kind == domain.TokenKindPasswordReset {
		return domain.ErrResetTokenInvalid
	}
	return domain.ErrConfirmationTokenInvalid
}

// devToken returns raw in a relaxed environment and "" everywhere else.
func (s *UserService) devToken(raw string) string {
	if IsRelaxedEnv(s.env) {
		return raw
	}
	return ""
}

func newToken(userID string, kind domain.TokenKind, now time.Time, ttl time.Duration) (domain.Token, error) {
	id, err := idgen.New()
	if err != nil {
		return domain.Token{}, fmt.Errorf("generating token: %w", err)
	}
	return domain.Token{
		ID:        id,
		UserID:    userID,
		Kind:      kind,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}, nil
}
