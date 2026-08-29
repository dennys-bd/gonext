package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golden-app/backend/users/domain"
)

func TestRequestPasswordReset_IssuesToken(t *testing.T) {
	f := newFixture(t, "dev")
	ctx := context.Background()
	user := f.registerAndConfirm(t, testEmail, testPassword)

	devToken, err := f.svc.RequestPasswordReset(ctx, "  ADA@example.com ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if devToken == "" {
		t.Fatal("expected a dev token outside production")
	}

	token, err := f.store.Tokens().Get(ctx, devToken)
	if err != nil {
		t.Fatalf("expected a token row: %v", err)
	}
	if token.Kind != domain.TokenKindPasswordReset || token.UserID != user.ID {
		t.Fatalf("unexpected token: %+v", token)
	}
}

func TestRequestPasswordReset_UnknownEmailIsSilent(t *testing.T) {
	f := newFixture(t, "dev")

	devToken, err := f.svc.RequestPasswordReset(context.Background(), "nobody@example.com")
	if err != nil {
		t.Fatalf("expected an unknown email not to error, got %v", err)
	}
	if devToken != "" {
		t.Fatalf("expected no token for an unknown email, got %q", devToken)
	}
	if len(f.notifier.Sent()) != 0 {
		t.Error("expected no notification for an unknown email")
	}
}

func TestRequestPasswordReset_RestrictedEnvWithholdsDevToken(t *testing.T) {
	for _, env := range []string{"stg", "prod"} {
		t.Run(env, func(t *testing.T) {
			f := newFixture(t, env)
			ctx := context.Background()
			f.registerAndConfirm(t, testEmail, testPassword)

			devToken, err := f.svc.RequestPasswordReset(ctx, testEmail)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if devToken != "" {
				t.Fatalf("expected no dev token in %q, got %q", env, devToken)
			}

			sent := f.notifier.Sent()
			if len(sent) == 0 || sent[len(sent)-1].Token == "" {
				t.Fatalf("expected the reset token to still reach the notifier in %q", env)
			}
		})
	}
}

func TestConfirmPasswordReset_ReplacesPassword(t *testing.T) {
	f := newFixture(t, "dev")
	ctx := context.Background()
	f.registerAndConfirm(t, testEmail, testPassword)

	devToken, err := f.svc.RequestPasswordReset(ctx, testEmail)
	if err != nil {
		t.Fatalf("request reset: %v", err)
	}

	const newPassword = "an entirely new password"
	if err := f.svc.ConfirmPasswordReset(ctx, devToken, newPassword); err != nil {
		t.Fatalf("confirm reset: %v", err)
	}

	if _, _, err := f.svc.Login(ctx, testEmail, newPassword); err != nil {
		t.Fatalf("expected the new password to work, got %v", err)
	}
	if _, _, err := f.svc.Login(ctx, testEmail, testPassword); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected the old password to stop working, got %v", err)
	}

	token, err := f.store.Tokens().Get(ctx, devToken)
	if err != nil {
		t.Fatalf("looking up token: %v", err)
	}
	if token.UsedAt == nil {
		t.Fatal("expected the reset token to be marked used")
	}
}

func TestConfirmPasswordReset_ShortPassword(t *testing.T) {
	f := newFixture(t, "dev")
	ctx := context.Background()
	f.registerAndConfirm(t, testEmail, testPassword)

	devToken, err := f.svc.RequestPasswordReset(ctx, testEmail)
	if err != nil {
		t.Fatalf("request reset: %v", err)
	}

	if err := f.svc.ConfirmPasswordReset(ctx, devToken, "short"); !errors.Is(err, domain.ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}

	token, err := f.store.Tokens().Get(ctx, devToken)
	if err != nil {
		t.Fatalf("looking up token: %v", err)
	}
	if token.UsedAt != nil {
		t.Fatal("expected a rejected password not to consume the token")
	}
}

func TestConfirmPasswordReset_InvalidToken(t *testing.T) {
	ctx := context.Background()
	const newPassword = "an entirely new password"

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
				f.registerAndConfirm(t, testEmail, testPassword)
				devToken, err := f.svc.RequestPasswordReset(ctx, testEmail)
				if err != nil {
					t.Fatalf("request reset: %v", err)
				}
				if err := f.svc.ConfirmPasswordReset(ctx, devToken, newPassword); err != nil {
					t.Fatalf("first confirm: %v", err)
				}
				return devToken
			},
		},
		{
			name: "expired",
			setup: func(t *testing.T, f *fixture) string {
				f.registerAndConfirm(t, testEmail, testPassword)
				devToken, err := f.svc.RequestPasswordReset(ctx, testEmail)
				if err != nil {
					t.Fatalf("request reset: %v", err)
				}
				token, err := f.store.Tokens().Get(ctx, devToken)
				if err != nil {
					t.Fatalf("looking up token: %v", err)
				}
				token.ExpiresAt = time.Now().UTC().Add(-time.Minute)
				if err := f.store.Tokens().Create(ctx, token); err != nil {
					t.Fatalf("expiring token: %v", err)
				}
				return devToken
			},
		},
		{
			name: "wrong kind",
			setup: func(t *testing.T, f *fixture) string {
				res, err := f.svc.Register(ctx, testEmail, testPassword)
				if err != nil {
					t.Fatalf("register: %v", err)
				}
				// A confirmation token must not authorise a password change.
				return res.DevToken
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, "dev")
			token := tt.setup(t, f)

			if err := f.svc.ConfirmPasswordReset(ctx, token, newPassword); !errors.Is(err, domain.ErrResetTokenInvalid) {
				t.Fatalf("expected ErrResetTokenInvalid, got %v", err)
			}
		})
	}
}
