package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	gonextauth "github.com/dennys-bd/gonext/auth"

	"golden-app/backend/users/domain"
	authadapter "golden-app/backend/users/internal/infrastructure/auth"
)

var errDatabaseDown = errors.New("database down")

// stubIssuer is a domain.SessionIssuer whose Validate outcome the
// test dictates.
type stubIssuer struct {
	identity gonextauth.Identity
	err      error
}

func (s stubIssuer) Issue(context.Context, string) (string, time.Time, error) {
	return "", time.Time{}, nil
}

func (s stubIssuer) Validate(context.Context, string) (gonextauth.Identity, domain.User, error) {
	if s.err != nil {
		return gonextauth.Identity{}, domain.User{}, s.err
	}
	return s.identity, domain.User{ID: s.identity.UserID}, nil
}

func (s stubIssuer) Revoke(context.Context, string) error { return nil }

func TestResolver_ValidSession(t *testing.T) {
	want := gonextauth.Identity{UserID: "u-1", Role: "admin"}
	resolver := authadapter.NewResolver(stubIssuer{identity: want})

	got, err := resolver.Resolve(context.Background(), "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UserID != want.UserID || got.Role != want.Role {
		t.Errorf("Resolve() = %+v, want %+v", got, want)
	}
}

func TestResolver_InvalidSessionBecomesErrUnauthenticated(t *testing.T) {
	resolver := authadapter.NewResolver(stubIssuer{err: domain.ErrSessionInvalid})

	_, err := resolver.Resolve(context.Background(), "token")
	if !errors.Is(err, gonextauth.ErrUnauthenticated) {
		t.Errorf("expected ErrUnauthenticated, got %v", err)
	}
}

// An infrastructure failure must NOT be translated: the middleware
// distinguishes the two to decide between 401 and 500, and mapping
// everything to ErrUnauthenticated would present an outage as a
// mass logout.
func TestResolver_InfrastructureErrorIsNotTranslated(t *testing.T) {
	resolver := authadapter.NewResolver(stubIssuer{err: errDatabaseDown})

	_, err := resolver.Resolve(context.Background(), "token")
	if errors.Is(err, gonextauth.ErrUnauthenticated) {
		t.Fatal("a database failure must not be reported as an invalid credential")
	}
	if !errors.Is(err, errDatabaseDown) {
		t.Errorf("expected the underlying error to be preserved, got %v", err)
	}
}
