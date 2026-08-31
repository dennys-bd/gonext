package application_test

import (
	"context"
	"errors"
	"testing"

	"golden-app/backend/users/domain"
)

func TestMe_ReturnsIdentityAndUser(t *testing.T) {
	f := newFixture(t, "dev")
	ctx := context.Background()
	user := f.registerAndConfirm(t, testEmail, testPassword)

	token, _, err := f.svc.Login(ctx, testEmail, testPassword)
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	identity, got, err := f.svc.Me(ctx, token)
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	if identity.UserID != user.ID || identity.Role != domain.RoleUser {
		t.Fatalf("unexpected identity: %+v", identity)
	}
	if got.ID != user.ID || got.Email != testEmail {
		t.Fatalf("unexpected user: %+v", got)
	}
	if !got.EmailVerified() {
		t.Error("expected the confirmed account to report a verified email")
	}
}

func TestMe_InvalidSession(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"missing token", ""},
		{"unknown token", "not-a-real-session"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, "dev")
			_, _, err := f.svc.Me(context.Background(), tt.token)
			if !errors.Is(err, domain.ErrSessionInvalid) {
				t.Fatalf("expected ErrSessionInvalid, got %v", err)
			}
		})
	}
}

func TestLogout_RevokesSession(t *testing.T) {
	f := newFixture(t, "dev")
	ctx := context.Background()
	f.registerAndConfirm(t, testEmail, testPassword)

	token, _, err := f.svc.Login(ctx, testEmail, testPassword)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := f.svc.Logout(ctx, token); err != nil {
		t.Fatalf("logout: %v", err)
	}

	if _, _, err := f.svc.Me(ctx, token); !errors.Is(err, domain.ErrSessionInvalid) {
		t.Fatalf("expected the revoked session to be invalid, got %v", err)
	}
}

func TestLogout_UnknownTokenIsNotAnError(t *testing.T) {
	f := newFixture(t, "dev")
	if err := f.svc.Logout(context.Background(), "never-issued"); err != nil {
		t.Fatalf("expected logging out an unknown session to be a no-op, got %v", err)
	}
}

func TestProfile_ReturnsTheAccount(t *testing.T) {
	f := newFixture(t, "test")
	user := f.registerAndConfirm(t, testEmail, testPassword)

	got, err := f.svc.Profile(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("Profile() returned user %q, want %q", got.ID, user.ID)
	}
}

func TestProfile_UnknownUser(t *testing.T) {
	f := newFixture(t, "test")

	if _, err := f.svc.Profile(context.Background(), "nope"); err == nil {
		t.Error("expected an error for an unknown user id")
	}
}
