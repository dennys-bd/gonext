// Package domain holds the users domain's public entities, value
// objects, domain errors, and ports. It depends on nothing outside
// the standard library.
package domain

import (
	"errors"
	"strings"
	"time"
)

// MinPasswordLength is the only password rule this domain enforces —
// length, no complexity requirements.
const MinPasswordLength = 8

// RoleUser is the role every self-registered account starts with.
const RoleUser = "user"

var (
	// ErrEmailRequired is returned when an email is missing or blank.
	ErrEmailRequired = errors.New("users: email is required")
	// ErrPasswordTooShort is returned when a password is shorter than MinPasswordLength.
	ErrPasswordTooShort = errors.New("users: password is too short")
	// ErrInvalidCredentials is returned for a wrong email or a wrong
	// password — deliberately the same error for both, so callers
	// cannot enumerate which emails are registered.
	ErrInvalidCredentials = errors.New("users: invalid credentials")
	// ErrEmailNotConfirmed is returned when a login is rejected because
	// the account's email has never been confirmed.
	ErrEmailNotConfirmed = errors.New("users: email is not confirmed")
	// ErrUserNotFound is returned when a User cannot be found.
	ErrUserNotFound = errors.New("users: user not found")
	// ErrEmailTaken is returned when registering an email that already
	// has an account. Registration deliberately reveals this — see the
	// design doc's registration section for the enumeration trade-off.
	ErrEmailTaken = errors.New("users: email already registered")
)

// User is a registered account.
type User struct {
	ID              string
	Email           string
	PasswordHash    string
	Role            string
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewUser constructs a User, validating that email is non-blank. It
// takes an already-hashed password: hashing is an application-layer
// concern, so the domain never sees a plaintext password.
func NewUser(id, email, passwordHash, role string, now time.Time) (User, error) {
	email = NormalizeEmail(email)
	if email == "" {
		return User{}, ErrEmailRequired
	}
	return User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// EmailVerified reports whether the account's email has been confirmed.
func (u User) EmailVerified() bool {
	return u.EmailVerifiedAt != nil
}

// NormalizeEmail lowercases and trims email so lookups and the
// uniqueness constraint agree on one canonical form.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidatePassword checks a plaintext password against the domain's
// minimum length rule.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	return nil
}
