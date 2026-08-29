package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"[PROJECT-NAME]/backend/users/domain"
)

func TestConfirmEmail_MarksVerifiedAndConsumesToken(t *testing.T) {
	f := newFixture(t, "dev")
	ctx := context.Background()

	res, err := f.svc.Register(ctx, testEmail, testPassword)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := f.svc.ConfirmEmail(ctx, res.DevToken); err != nil {
		t.Fatalf("confirm email: %v", err)
	}

	user, err := f.store.Users().GetByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("looking up user: %v", err)
	}
	if !user.EmailVerified() {
		t.Fatal("expected the email to be verified")
	}

	token, err := f.store.Tokens().Get(ctx, res.DevToken)
	if err != nil {
		t.Fatalf("looking up token: %v", err)
	}
	if token.UsedAt == nil {
		t.Fatal("expected the confirmation token to be marked used")
	}
}

func TestConfirmEmail_InvalidToken(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		setup func(t *testing.T, f *fixture) string
	}{
		{
			name:  "unknown token",
			setup: func(*testing.T, *fixture) string { return "not-a-real-token" },
		},
		{
			name: "already consumed",
			setup: func(t *testing.T, f *fixture) string {
				res, err := f.svc.Register(ctx, testEmail, testPassword)
				if err != nil {
					t.Fatalf("register: %v", err)
				}
				if err := f.svc.ConfirmEmail(ctx, res.DevToken); err != nil {
					t.Fatalf("first confirm: %v", err)
				}
				return res.DevToken
			},
		},
		{
			name: "expired",
			setup: func(t *testing.T, f *fixture) string {
				res, err := f.svc.Register(ctx, testEmail, testPassword)
				if err != nil {
					t.Fatalf("register: %v", err)
				}
				token, err := f.store.Tokens().Get(ctx, res.DevToken)
				if err != nil {
					t.Fatalf("looking up token: %v", err)
				}
				token.ExpiresAt = time.Now().UTC().Add(-time.Minute)
				if err := f.store.Tokens().Create(ctx, token); err != nil {
					t.Fatalf("expiring token: %v", err)
				}
				return res.DevToken
			},
		},
		{
			name: "wrong kind",
			setup: func(t *testing.T, f *fixture) string {
				f.registerAndConfirm(t, testEmail, testPassword)
				// A password_reset token must not confirm an email.
				devToken, err := f.svc.RequestPasswordReset(ctx, testEmail)
				if err != nil {
					t.Fatalf("request reset: %v", err)
				}
				return devToken
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, "dev")
			token := tt.setup(t, f)

			if err := f.svc.ConfirmEmail(ctx, token); !errors.Is(err, domain.ErrConfirmationTokenInvalid) {
				t.Fatalf("expected ErrConfirmationTokenInvalid, got %v", err)
			}
		})
	}
}
