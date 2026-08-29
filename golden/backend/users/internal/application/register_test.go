package application_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"golden-app/backend/users/domain"
)

func TestRegister_NewEmail(t *testing.T) {
	f := newFixture(t, "dev")
	ctx := context.Background()

	res, err := f.svc.Register(ctx, "  Ada@Example.COM ", testPassword)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.DevToken == "" {
		t.Fatal("expected a dev token in a relaxed environment")
	}

	user, err := f.store.Users().GetByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("expected the user to have been created: %v", err)
	}
	if user.Email != testEmail {
		t.Errorf("expected normalized email %q, got %q", testEmail, user.Email)
	}
	if user.Role != domain.RoleUser {
		t.Errorf("expected role %q, got %q", domain.RoleUser, user.Role)
	}
	if user.EmailVerified() {
		t.Error("expected a newly registered user to be unconfirmed")
	}
	if user.PasswordHash == testPassword {
		t.Error("expected the password to be stored hashed")
	}

	token, err := f.store.Tokens().Get(ctx, res.DevToken)
	if err != nil {
		t.Fatalf("expected a token row for the dev token: %v", err)
	}
	if token.Kind != domain.TokenKindEmailConfirmation {
		t.Errorf("expected an email_confirmation token, got %q", token.Kind)
	}

	sent := f.notifier.Sent()
	if len(sent) != 1 || sent[0].Kind != "confirmation" {
		t.Fatalf("expected one confirmation notification, got %+v", sent)
	}
	if sent[0].Token != res.DevToken {
		t.Error("expected the notification to carry the confirmation token")
	}
}

func TestRegister_ExistingEmail(t *testing.T) {
	f := newFixture(t, "dev")
	ctx := context.Background()

	if _, err := f.svc.Register(ctx, testEmail, testPassword); err != nil {
		t.Fatalf("first register: %v", err)
	}
	original, err := f.store.Users().GetByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("looking up first user: %v", err)
	}

	_, err = f.svc.Register(ctx, "  ADA@example.com ", "an entirely different password")
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}

	after, err := f.store.Users().GetByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("looking up user after the rejected register: %v", err)
	}
	if after.ID != original.ID || after.PasswordHash != original.PasswordHash {
		t.Fatalf("expected the existing account to be untouched, got %+v", after)
	}
	if len(f.notifier.Sent()) != 1 {
		t.Fatalf("expected no second notification, got %+v", f.notifier.Sent())
	}
}

// Registration must not check-then-act: the uniqueness decision
// belongs to the store's constraint, so exactly one of N concurrent
// registrations of the same address can win.
func TestRegister_ConcurrentSameEmail(t *testing.T) {
	const racers = 8

	f := newFixture(t, "dev")
	ctx := context.Background()

	var (
		start   sync.WaitGroup
		done    sync.WaitGroup
		mu      sync.Mutex
		wins    int
		taken   int
		unknown []error
	)
	start.Add(1)
	done.Add(racers)

	for range racers {
		go func() {
			defer done.Done()
			start.Wait()

			_, err := f.svc.Register(ctx, testEmail, testPassword)

			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				wins++
			case errors.Is(err, domain.ErrEmailTaken):
				taken++
			default:
				unknown = append(unknown, err)
			}
		}()
	}

	start.Done()
	done.Wait()

	if len(unknown) != 0 {
		t.Fatalf("expected only nil or ErrEmailTaken, got %v", unknown)
	}
	if wins != 1 {
		t.Fatalf("expected exactly 1 winner, got %d (and %d rejected)", wins, taken)
	}
	if taken != racers-1 {
		t.Fatalf("expected %d rejections, got %d", racers-1, taken)
	}

	if _, err := f.store.Users().GetByEmail(ctx, testEmail); err != nil {
		t.Fatalf("expected exactly one account to exist: %v", err)
	}
}

func TestRegister_RestrictedEnvWithholdsDevToken(t *testing.T) {
	for _, env := range []string{"stg", "prod"} {
		t.Run(env, func(t *testing.T) {
			f := newFixture(t, env)

			res, err := f.svc.Register(context.Background(), testEmail, testPassword)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.DevToken != "" {
				t.Fatalf("expected no dev token in %q, got %q", env, res.DevToken)
			}
			if len(f.notifier.Sent()) != 1 {
				t.Fatalf("expected the confirmation to still be sent in %q", env)
			}
		})
	}
}

func TestRegister_ValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{"blank email", "   ", testPassword, domain.ErrEmailRequired},
		{"short password", testEmail, "short", domain.ErrPasswordTooShort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, "dev")
			_, err := f.svc.Register(context.Background(), tt.email, tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if len(f.notifier.Sent()) != 0 {
				t.Error("expected no notification for an invalid registration")
			}
		})
	}
}
