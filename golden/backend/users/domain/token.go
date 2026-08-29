package domain

import (
	"errors"
	"time"
)

// TokenKind distinguishes the one-shot token flows that share the
// tokens table.
type TokenKind string

const (
	// TokenKindEmailConfirmation confirms a newly registered email address.
	TokenKindEmailConfirmation TokenKind = "email_confirmation"
	// TokenKindPasswordReset authorises setting a new password.
	TokenKindPasswordReset TokenKind = "password_reset"
)

var (
	// ErrConfirmationTokenInvalid is returned for an unknown, expired,
	// or already-consumed email confirmation token.
	ErrConfirmationTokenInvalid = errors.New("users: confirmation token is invalid")
	// ErrResetTokenInvalid is returned for an unknown, expired, or
	// already-consumed password reset token.
	ErrResetTokenInvalid = errors.New("users: reset token is invalid")
	// ErrTokenNotFound is returned when a Token cannot be found by id.
	ErrTokenNotFound = errors.New("users: token not found")
)

// Token is a single-use, expiring credential emailed to a user.
type Token struct {
	ID        string
	UserID    string
	Kind      TokenKind
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

// Usable reports whether t is of kind and is neither consumed nor
// expired as of now.
func (t Token) Usable(kind TokenKind, now time.Time) bool {
	return t.Kind == kind && t.UsedAt == nil && now.Before(t.ExpiresAt)
}
