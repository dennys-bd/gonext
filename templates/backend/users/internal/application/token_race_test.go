package application_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"[PROJECT-NAME]/backend/users/domain"
)

// Both one-shot flows read the token before opening the transaction,
// so a concurrent consumer can win the MarkUsed race. The loser must
// see the flow's documented "invalid token" sentinel — which the HTTP
// layer renders as 400 — not a wrapped repository error that would
// escape as a 500.
func TestConfirmEmail_ConcurrentConsumeYieldsInvalidToken(t *testing.T) {
	f := newFixture(t, "dev")
	ctx := context.Background()

	res, err := f.svc.Register(ctx, testEmail, testPassword)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	wins, losses := raceConsumers(t, 6, func() error {
		return f.svc.ConfirmEmail(ctx, res.DevToken)
	}, domain.ErrConfirmationTokenInvalid)

	if wins != 1 {
		t.Fatalf("expected exactly 1 successful confirmation, got %d", wins)
	}
	if losses != 5 {
		t.Fatalf("expected 5 invalid-token results, got %d", losses)
	}

	user, err := f.store.Users().GetByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("looking up user: %v", err)
	}
	if !user.EmailVerified() {
		t.Fatal("expected the winning confirmation to have verified the email")
	}
}

func TestConfirmPasswordReset_ConcurrentConsumeYieldsInvalidToken(t *testing.T) {
	f := newFixture(t, "dev")
	ctx := context.Background()
	f.registerAndConfirm(t, testEmail, testPassword)

	devToken, err := f.svc.RequestPasswordReset(ctx, testEmail)
	if err != nil {
		t.Fatalf("request reset: %v", err)
	}

	const newPassword = "an entirely new password"
	wins, losses := raceConsumers(t, 6, func() error {
		return f.svc.ConfirmPasswordReset(ctx, devToken, newPassword)
	}, domain.ErrResetTokenInvalid)

	if wins != 1 {
		t.Fatalf("expected exactly 1 successful reset, got %d", wins)
	}
	if losses != 5 {
		t.Fatalf("expected 5 invalid-token results, got %d", losses)
	}

	if _, _, err := f.svc.Login(ctx, testEmail, newPassword); err != nil {
		t.Fatalf("expected the winning reset to have applied the new password, got %v", err)
	}
}

// Sequential re-consumption must report the same sentinel as the
// concurrent loser — one code path, not two.
func TestConfirmEmail_SecondConsumeYieldsInvalidToken(t *testing.T) {
	f := newFixture(t, "dev")
	ctx := context.Background()

	res, err := f.svc.Register(ctx, testEmail, testPassword)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := f.svc.ConfirmEmail(ctx, res.DevToken); err != nil {
		t.Fatalf("first confirm: %v", err)
	}

	if err := f.svc.ConfirmEmail(ctx, res.DevToken); !errors.Is(err, domain.ErrConfirmationTokenInvalid) {
		t.Fatalf("expected ErrConfirmationTokenInvalid, got %v", err)
	}
}

// raceConsumers runs consume in n goroutines released together,
// returning how many succeeded and how many returned wantErr. Any
// other error fails the test.
func raceConsumers(t *testing.T, n int, consume func() error, wantErr error) (wins, losses int) {
	t.Helper()

	var (
		start sync.WaitGroup
		done  sync.WaitGroup
		mu    sync.Mutex
		other []error
	)
	start.Add(1)
	done.Add(n)

	for range n {
		go func() {
			defer done.Done()
			start.Wait()

			err := consume()

			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				wins++
			case errors.Is(err, wantErr):
				losses++
			default:
				other = append(other, err)
			}
		}()
	}

	start.Done()
	done.Wait()

	if len(other) != 0 {
		t.Fatalf("expected only nil or %v, got %v", wantErr, other)
	}
	return wins, losses
}
