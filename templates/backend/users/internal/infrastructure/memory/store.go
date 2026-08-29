// Package memory provides in-memory adapters for the users domain's
// ports, so application-layer use cases can be tested without a
// database.
package memory

import (
	"context"
	"sync"
	"time"

	"[PROJECT-NAME]/backend/users/domain"
)

var (
	_ domain.Store           = (*Store)(nil)
	_ domain.TxRunner        = (*Store)(nil)
	_ domain.UserRepository  = (*UserRepository)(nil)
	_ domain.TokenRepository = (*TokenRepository)(nil)
)

// Store is an in-memory domain.Store. It also satisfies
// domain.TxRunner by running fn directly against itself: the fakes
// have no rollback semantics, which is enough for use-case tests that
// assert on behaviour rather than on transactional isolation.
type Store struct {
	users  *UserRepository
	tokens *TokenRepository
}

// NewStore constructs an empty in-memory Store.
func NewStore() *Store {
	return &Store{users: NewUserRepository(), tokens: NewTokenRepository()}
}

// Users returns the in-memory user repository.
func (s *Store) Users() domain.UserRepository { return s.users }

// Tokens returns the in-memory token repository.
func (s *Store) Tokens() domain.TokenRepository { return s.tokens }

// RunInTx runs fn against s. There is no rollback: a failed fn leaves
// behind whatever it already wrote.
func (s *Store) RunInTx(ctx context.Context, fn func(context.Context, domain.Store) error) error {
	return fn(ctx, s)
}

// UserRepository is an in-memory, mutex-guarded domain.UserRepository.
type UserRepository struct {
	mu    sync.Mutex
	users map[string]domain.User
}

// NewUserRepository constructs an empty in-memory UserRepository.
func NewUserRepository() *UserRepository {
	return &UserRepository{users: make(map[string]domain.User)}
}

// Create stores u, keyed by its ID, or returns
// domain.ErrEmailTaken if another account already uses its email —
// mirroring the unique constraint the Postgres adapter relies on.
func (r *UserRepository) Create(_ context.Context, u domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.users {
		if existing.Email == u.Email && existing.ID != u.ID {
			return domain.ErrEmailTaken
		}
	}
	r.users[u.ID] = u
	return nil
}

// GetByID retrieves the User with the given id, or domain.ErrUserNotFound.
func (r *UserRepository) GetByID(_ context.Context, id string) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

// GetByEmail retrieves the User with the given email, or domain.ErrUserNotFound.
func (r *UserRepository) GetByEmail(_ context.Context, email string) (domain.User, error) {
	email = domain.NormalizeEmail(email)
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

// UpdatePasswordHash replaces the stored password hash for id.
func (r *UserRepository) UpdatePasswordHash(_ context.Context, id, passwordHash string, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.PasswordHash, u.UpdatedAt = passwordHash, updatedAt
	r.users[id] = u
	return nil
}

// MarkEmailVerified records verifiedAt as the confirmation time for id.
func (r *UserRepository) MarkEmailVerified(_ context.Context, id string, verifiedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.EmailVerifiedAt, u.UpdatedAt = &verifiedAt, verifiedAt
	r.users[id] = u
	return nil
}

// TokenRepository is an in-memory, mutex-guarded domain.TokenRepository.
type TokenRepository struct {
	mu     sync.Mutex
	tokens map[string]domain.Token
}

// NewTokenRepository constructs an empty in-memory TokenRepository.
func NewTokenRepository() *TokenRepository {
	return &TokenRepository{tokens: make(map[string]domain.Token)}
}

// Create stores t, keyed by its ID.
func (r *TokenRepository) Create(_ context.Context, t domain.Token) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[t.ID] = t
	return nil
}

// Get retrieves the Token with the given id, or domain.ErrTokenNotFound.
func (r *TokenRepository) Get(_ context.Context, id string) (domain.Token, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tokens[id]
	if !ok {
		return domain.Token{}, domain.ErrTokenNotFound
	}
	return t, nil
}

// MarkUsed records usedAt as the consumption time for id. Like the
// Postgres adapter it only matches an unconsumed token, so a second
// consumption reports domain.ErrTokenNotFound rather than silently
// succeeding.
func (r *TokenRepository) MarkUsed(_ context.Context, id string, usedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tokens[id]
	if !ok || t.UsedAt != nil {
		return domain.ErrTokenNotFound
	}
	t.UsedAt = &usedAt
	r.tokens[id] = t
	return nil
}
