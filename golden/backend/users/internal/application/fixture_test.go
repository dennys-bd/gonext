package application_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"golden-app/backend/users/domain"
	"golden-app/backend/users/internal/application"
	"golden-app/backend/users/internal/infrastructure/memory"
)

const (
	testEmail    = "ada@example.com"
	testPassword = "correct horse battery staple"
)

// countingHasher is a deliberately trivial domain.PasswordHasher that
// records how often it was called. Service tests use it instead of
// Argon2id so they stay fast and can assert on hashing behaviour that
// is otherwise invisible — notably that both registration branches
// hash, which is what keeps them indistinguishable by timing.
type countingHasher struct {
	mu           sync.Mutex
	hashCalls    int
	verifyCalls  int
	hashesPrefix string
}

func newCountingHasher() *countingHasher {
	return &countingHasher{hashesPrefix: "hashed:"}
}

func (h *countingHasher) Hash(password string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hashCalls++
	return h.hashesPrefix + password, nil
}

func (h *countingHasher) Verify(password, hash string) (bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.verifyCalls++
	return hash == h.hashesPrefix+password, nil
}

func (h *countingHasher) counts() (hashes, verifies int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hashCalls, h.verifyCalls
}

type fixture struct {
	svc      *application.UserService
	store    *memory.Store
	issuer   *memory.SessionIssuer
	notifier *memory.Notifier
	hasher   *countingHasher
}

func newFixture(t *testing.T, env string) *fixture {
	t.Helper()

	store := memory.NewStore()
	issuer := memory.NewSessionIssuer(store.Users(), time.Hour)
	notifier := memory.NewNotifier()
	hasher := newCountingHasher()

	return &fixture{
		svc:      application.NewUserService(store, store, issuer, notifier, hasher, env),
		store:    store,
		issuer:   issuer,
		notifier: notifier,
		hasher:   hasher,
	}
}

// registerAndConfirm registers email and consumes the confirmation
// token so the account can log in under any Env. It reads the token
// off the notifier rather than the Register result, since production
// withholds the dev token.
func (f *fixture) registerAndConfirm(t *testing.T, email, password string) domain.User {
	t.Helper()
	ctx := context.Background()

	if _, err := f.svc.Register(ctx, email, password); err != nil {
		t.Fatalf("register: %v", err)
	}
	sent := f.notifier.Sent()
	if err := f.svc.ConfirmEmail(ctx, sent[len(sent)-1].Token); err != nil {
		t.Fatalf("confirm email: %v", err)
	}

	user, err := f.store.Users().GetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("looking up registered user: %v", err)
	}
	return user
}
