package presentation

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"

	"[PROJECT-NAME]/backend/users/domain"
	"[PROJECT-NAME]/backend/users/internal/application"
	"[PROJECT-NAME]/backend/users/internal/infrastructure/memory"
)

// secretDetail stands in for the kind of text an unexpected error
// carries in production — a driver message naming tables, columns, or
// the host. None of it may reach the client.
// Kept free of quotes so the assertion below can look for it
// verbatim in slog's escaped output.
const secretDetail = "pq: relation users_email_key does not exist on host db-primary-01"

// failingStore is a domain.Store whose user repository always fails
// with an error the users group has no mapping for.
type failingStore struct {
	domain.Store
	users failingUserRepository
}

func (s *failingStore) Users() domain.UserRepository { return s.users }

type failingUserRepository struct {
	domain.UserRepository
}

func (failingUserRepository) GetByEmail(context.Context, string) (domain.User, error) {
	return domain.User{}, errors.New(secretDetail)
}

// The generic mapped/unmapped/ordering behaviour is proven once in
// httpx/errors_test.go; this is the per-track assertion that the
// users group is actually wired with a logger and does not leak.
func TestUnhandledError_IsLoggedAndNotLeaked(t *testing.T) {
	store := &failingStore{}
	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logs, nil))

	svc := application.NewUserService(
		store,
		memory.NewStore(),
		memory.NewSessionIssuer(memory.NewUserRepository(), time.Hour),
		memory.NewNotifier(),
		application.NewArgon2Hasher(),
		"test",
	)

	_, api := humatest.New(t)
	RegisterUsers(api, svc, NewCookieOptions("test"), logger)

	resp := api.Post("/users/login", map[string]string{"email": testEmail, "password": testPassword})
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	if strings.Contains(body, secretDetail) {
		t.Fatalf("expected the driver message not to reach the client, got %s", body)
	}
	if !strings.Contains(logs.String(), secretDetail) {
		t.Fatalf("expected the real error to be logged server-side, got %q", logs.String())
	}
}
