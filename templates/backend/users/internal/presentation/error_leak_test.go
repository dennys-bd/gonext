package presentation

import (
	"bytes"
	"context"
	"encoding/json"
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
// with an error the presentation layer has no mapping for.
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

// huma renders a non-StatusError by copying err.Error() into the
// response body, so an unhandled error must be replaced before it is
// returned — not merely left for huma to turn into a 500.
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
	if strings.Contains(body, "users:") || strings.Contains(body, "looking up user") {
		t.Fatalf("expected no wrapped internals in the body, got %s", body)
	}

	// Explicitly check the field huma fills from err.Error().
	var rendered struct {
		Detail string `json:"detail"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	decode(t, resp.Body.Bytes(), &rendered)
	if rendered.Detail != "internal server error" {
		t.Fatalf("expected a flat detail, got %q", rendered.Detail)
	}
	for _, e := range rendered.Errors {
		if strings.Contains(e.Message, secretDetail) {
			t.Fatalf("expected no leak in errors[].message, got %q", e.Message)
		}
	}

	if !strings.Contains(logs.String(), secretDetail) {
		t.Fatalf("expected the real error to be logged server-side, got %q", logs.String())
	}
}

// The mapped domain errors keep their own messages — the generic
// replacement applies only to errors with no mapping.
func TestMappedError_KeepsItsDetail(t *testing.T) {
	api := newTestAPI(t, "test")

	resp := api.Post("/users/confirm-email", map[string]string{"token": "not-a-real-token"})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}

	var body struct {
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if body.Detail != domain.ErrConfirmationTokenInvalid.Error() {
		t.Fatalf("expected the domain message, got %q", body.Detail)
	}
}
