package api_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dennys-bd/gonext/auth"

	"golden-app/backend/internal/presentation/api"
	"golden-app/backend/internal/presentation/httpx"
)

// errBackendDown stands in for an infrastructure failure inside a
// Resolver — a database outage, not a bad credential.
var errBackendDown = errors.New("backend down")

// fakeResolver resolves exactly one token and fails everything else
// the way a real provider would.
type fakeResolver struct {
	token    string
	identity auth.Identity
	err      error
}

func (f fakeResolver) Resolve(_ context.Context, token string) (auth.Identity, error) {
	if f.err != nil {
		return auth.Identity{}, f.err
	}
	if token != f.token {
		return auth.Identity{}, auth.ErrUnauthenticated
	}
	return f.identity, nil
}

type probeOutput struct {
	Body struct {
		Seen   bool   `json:"seen"`
		UserID string `json:"userId"`
	}
}

// newProbeAPI mounts one operation with the given security rule,
// guarded by the middleware backed by resolver.
func newProbeAPI(t *testing.T, security []map[string][]string, resolver auth.Resolver) humatest.TestAPI {
	t.Helper()

	_, testAPI := humatest.New(t)
	testAPI.UseMiddleware(api.NewAuthMiddleware(
		testAPI,
		resolver,
		api.ProvideAuthConfig(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	))

	httpx.Register(testAPI, huma.Operation{
		OperationID: "probe",
		Method:      http.MethodGet,
		Path:        "/probe",
		Security:    security,
	}, func(ctx *httpx.Ctx, _ *struct{}) (*probeOutput, error) {
		out := &probeOutput{}
		identity, ok := ctx.IdentityOK()
		out.Body.Seen, out.Body.UserID = ok, identity.UserID
		return out, nil
	})

	return testAPI
}

func TestAuthMiddleware(t *testing.T) {
	admin := auth.Identity{UserID: "u-1", Role: "admin", Permissions: []string{"posts:delete"}}
	good := fakeResolver{token: "valid", identity: admin}

	tests := []struct {
		name       string
		security   []map[string][]string
		resolver   auth.Resolver
		cookie     string
		wantStatus int
		wantSeen   bool
	}{
		{
			name:       "undeclared operation ignores a credential entirely",
			security:   nil,
			resolver:   good,
			cookie:     "valid",
			wantStatus: http.StatusOK,
			wantSeen:   false,
		},
		{
			name:       "required with a valid credential injects the identity",
			security:   auth.Required(),
			resolver:   good,
			cookie:     "valid",
			wantStatus: http.StatusOK,
			wantSeen:   true,
		},
		{
			name:       "required with no credential is 401",
			security:   auth.Required(),
			resolver:   good,
			cookie:     "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "required with an invalid credential is 401",
			security:   auth.Required(),
			resolver:   good,
			cookie:     "garbage",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "optional with no credential proceeds anonymously",
			security:   auth.Optional(),
			resolver:   good,
			cookie:     "",
			wantStatus: http.StatusOK,
			wantSeen:   false,
		},
		{
			name:       "optional with an invalid credential proceeds anonymously",
			security:   auth.Optional(),
			resolver:   good,
			cookie:     "garbage",
			wantStatus: http.StatusOK,
			wantSeen:   false,
		},
		{
			name:       "optional with a valid credential injects the identity",
			security:   auth.Optional(),
			resolver:   good,
			cookie:     "valid",
			wantStatus: http.StatusOK,
			wantSeen:   true,
		},
		{
			name:       "matching role is allowed",
			security:   auth.RequireRole("admin"),
			resolver:   good,
			cookie:     "valid",
			wantStatus: http.StatusOK,
			wantSeen:   true,
		},
		{
			name:       "wrong role is 403",
			security:   auth.RequireRole("superuser"),
			resolver:   good,
			cookie:     "valid",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "held permission is allowed",
			security:   auth.RequirePermission("posts:delete"),
			resolver:   good,
			cookie:     "valid",
			wantStatus: http.StatusOK,
			wantSeen:   true,
		},
		{
			name:       "missing permission is 403",
			security:   auth.RequirePermission("posts:publish"),
			resolver:   good,
			cookie:     "valid",
			wantStatus: http.StatusForbidden,
		},
		{
			// The row that matters most: an outage must not read as
			// "your session expired".
			name:       "resolver infrastructure failure is 500, not 401",
			security:   auth.Required(),
			resolver:   fakeResolver{err: errBackendDown},
			cookie:     "valid",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			probeAPI := newProbeAPI(t, tt.security, tt.resolver)

			var resp *httptest.ResponseRecorder
			if tt.cookie == "" {
				resp = probeAPI.Get("/probe")
			} else {
				resp = probeAPI.Get("/probe", fmt.Sprintf("Cookie: %s=%s", auth.DefaultCookieName, tt.cookie))
			}

			if resp.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tt.wantStatus, resp.Code, resp.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				return
			}

			wantFragment := `"seen":false`
			if tt.wantSeen {
				wantFragment = `"seen":true`
			}
			if body := resp.Body.String(); !strings.Contains(body, wantFragment) {
				t.Errorf("expected body to contain %s, got %s", wantFragment, body)
			}
		})
	}
}
