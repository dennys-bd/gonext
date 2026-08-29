package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"[PROJECT-NAME]/backend/users/domain"
)

func TestLogin_IssuesSession(t *testing.T) {
	f := newFixture(t, "dev")
	ctx := context.Background()
	user := f.registerAndConfirm(t, testEmail, testPassword)

	token, expiresAt, err := f.svc.Login(ctx, "  ADA@example.com ", testPassword)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty session token")
	}
	if !expiresAt.After(time.Now().UTC()) {
		t.Fatalf("expected a future expiry, got %s", expiresAt)
	}

	identity, _, err := f.issuer.Validate(ctx, token)
	if err != nil {
		t.Fatalf("validating the issued session: %v", err)
	}
	if identity.UserID != user.ID {
		t.Fatalf("expected the session to resolve to %q, got %q", user.ID, identity.UserID)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
	}{
		{"unknown email", "nobody@example.com", testPassword},
		{"wrong password", testEmail, "not the right password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, "dev")
			f.registerAndConfirm(t, testEmail, testPassword)

			_, _, err := f.svc.Login(context.Background(), tt.email, tt.password)
			if !errors.Is(err, domain.ErrInvalidCredentials) {
				t.Fatalf("expected ErrInvalidCredentials, got %v", err)
			}
		})
	}
}

// An unknown email must cost a password hash too, so it cannot be
// distinguished from a known email with a wrong password by timing.
func TestLogin_UnknownEmailStillHashes(t *testing.T) {
	f := newFixture(t, "dev")

	before, _ := f.hasher.counts()
	_, _, err := f.svc.Login(context.Background(), "nobody@example.com", testPassword)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	after, _ := f.hasher.counts()

	if after != before+1 {
		t.Fatalf("expected one hash on the unknown-email branch, got %d", after-before)
	}
}

func TestLogin_UnconfirmedEmail(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		wantErr error
	}{
		{"blocked in prod", "prod", domain.ErrEmailNotConfirmed},
		{"blocked in stg", "stg", domain.ErrEmailNotConfirmed},
		{"allowed in dev", "dev", nil},
		{"allowed in test", "test", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, tt.env)
			ctx := context.Background()

			if _, err := f.svc.Register(ctx, testEmail, testPassword); err != nil {
				t.Fatalf("register: %v", err)
			}

			_, _, err := f.svc.Login(ctx, testEmail, testPassword)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected login to succeed, got %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestLogin_ConfirmedEmailInRestrictedEnv(t *testing.T) {
	for _, env := range []string{"stg", "prod"} {
		t.Run(env, func(t *testing.T) {
			f := newFixture(t, env)
			ctx := context.Background()

			res, err := f.svc.Register(ctx, testEmail, testPassword)
			if err != nil {
				t.Fatalf("register: %v", err)
			}
			if res.DevToken != "" {
				t.Fatalf("expected no dev token in %q", env)
			}
			// A restricted env withholds the dev token, so drive the
			// confirmation through the notification, as a mailer would.
			if err := f.svc.ConfirmEmail(ctx, f.notifier.Sent()[0].Token); err != nil {
				t.Fatalf("confirm email: %v", err)
			}

			if _, _, err := f.svc.Login(ctx, testEmail, testPassword); err != nil {
				t.Fatalf("expected a confirmed user to log in under %q, got %v", env, err)
			}
		})
	}
}
