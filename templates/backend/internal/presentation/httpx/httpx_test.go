package httpx_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dennys-bd/gonext/auth"

	"[PROJECT-NAME]/backend/internal/presentation/httpx"
)

type emptyInput struct{}

type idOutput struct {
	Body struct {
		UserID string `json:"userId"`
		Seen   bool   `json:"seen"`
	}
}

// Register must hand the handler a *Ctx carrying whatever identity is
// already in the request context — that is the whole contract between
// the middleware and a handler.
func TestRegister_PassesInjectedIdentity(t *testing.T) {
	_, api := humatest.New(t)

	// Stand in for the auth middleware.
	api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithValue(ctx, auth.ContextKey, auth.Identity{UserID: "u-1", Role: "admin"}))
	})

	httpx.Register(api, huma.Operation{
		OperationID: "probe",
		Method:      http.MethodGet,
		Path:        "/probe",
	}, func(ctx *httpx.Ctx, _ *emptyInput) (*idOutput, error) {
		out := &idOutput{}
		out.Body.UserID = ctx.Identity().UserID
		_, out.Body.Seen = ctx.IdentityOK()
		return out, nil
	})

	resp := api.Get("/probe")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, `"userId":"u-1"`) || !strings.Contains(body, `"seen":true`) {
		t.Errorf("unexpected body: %s", body)
	}
}

// *Ctx must satisfy context.Context so handlers can pass it straight
// to services and repositories without unwrapping.
func TestCtx_SatisfiesContext(t *testing.T) {
	var _ context.Context = (*httpx.Ctx)(nil)

	_, api := humatest.New(t)
	httpx.Register(api, huma.Operation{
		OperationID: "passthrough",
		Method:      http.MethodGet,
		Path:        "/passthrough",
	}, func(ctx *httpx.Ctx, _ *emptyInput) (*idOutput, error) {
		// Compiles only if *Ctx is usable wherever context.Context is.
		if err := takesContext(ctx); err != nil {
			return nil, err
		}
		out := &idOutput{}
		_, out.Body.Seen = ctx.IdentityOK()
		return out, nil
	})

	resp := api.Get("/passthrough")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if body := resp.Body.String(); !strings.Contains(body, `"seen":false`) {
		t.Errorf("expected no identity, got: %s", body)
	}
}

func TestCtx_IdentityPanicsWhenAbsent(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected Identity() to panic when no identity was injected")
		}
	}()

	httpx.NewCtx(context.Background()).Identity()
}

func takesContext(_ context.Context) error { return nil }
