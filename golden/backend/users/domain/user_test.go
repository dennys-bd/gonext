package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"golden-app/backend/users/domain"
)

func TestNewUser_NormalizesEmail(t *testing.T) {
	now := time.Now().UTC()

	u, err := domain.NewUser("u-1", "  Ada@Example.COM ", "hash", domain.RoleUser, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Email != "ada@example.com" {
		t.Errorf("expected normalized email, got %q", u.Email)
	}
	if !u.CreatedAt.Equal(now) || !u.UpdatedAt.Equal(now) {
		t.Errorf("expected CreatedAt and UpdatedAt to be %s, got %s / %s", now, u.CreatedAt, u.UpdatedAt)
	}
	if u.EmailVerified() {
		t.Error("expected a new user to be unverified")
	}
}

func TestNewUser_BlankEmail(t *testing.T) {
	for _, email := range []string{"", "   "} {
		_, err := domain.NewUser("u-1", email, "hash", domain.RoleUser, time.Now().UTC())
		if !errors.Is(err, domain.ErrEmailRequired) {
			t.Fatalf("email %q: expected ErrEmailRequired, got %v", email, err)
		}
	}
}

func TestUser_EmailVerified(t *testing.T) {
	verifiedAt := time.Now().UTC()
	u := domain.User{EmailVerifiedAt: &verifiedAt}
	if !u.EmailVerified() {
		t.Error("expected EmailVerified to be true when EmailVerifiedAt is set")
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{"empty", "", domain.ErrPasswordTooShort},
		{"one short of the minimum", strings.Repeat("a", domain.MinPasswordLength-1), domain.ErrPasswordTooShort},
		{"exactly the minimum", strings.Repeat("a", domain.MinPasswordLength), nil},
		{"comfortably long", "correct horse battery staple", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidatePassword(tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}
