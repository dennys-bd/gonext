package httpx_test

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"

	"[PROJECT-NAME]/backend/internal/presentation/httpx"
)

var errGroupSentinel = errors.New("group: sentinel")

type groupInput struct{}

type groupOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func okHandler(_ *httpx.Ctx, _ *groupInput) (*groupOutput, error) {
	out := &groupOutput{}
	out.Body.OK = true
	return out, nil
}

func TestGroup_ComposesPath(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		path   string
		want   string
	}{
		{"collection root passes the empty suffix", "/stubs", "", "/stubs"},
		{"a suffix appends to the prefix", "/stubs", "/{id}", "/stubs/{id}"},
		{"an empty prefix leaves the path untouched", "", "/healthz", "/healthz"},
		// "/" is not a synonym for "": pinned so nobody trims it away.
		{"a slash suffix keeps its trailing slash", "/stubs", "/", "/stubs/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)
			g := httpx.NewGroup(api, tt.prefix, "Example", discardLogger())

			httpx.Get(g, tt.path, "probe", okHandler)

			item, ok := api.OpenAPI().Paths[tt.want]
			if !ok {
				t.Fatalf("no operation registered at %q; paths: %v", tt.want, registeredPaths(api.OpenAPI().Paths))
			}
			if item.Get == nil {
				t.Fatalf("path %q has no GET operation", tt.want)
			}
			if item.Get.Path != tt.want {
				t.Errorf("operation path = %q, want %q", item.Get.Path, tt.want)
			}
		})
	}
}

func TestGroup_AppliesExactlyItsTag(t *testing.T) {
	_, api := humatest.New(t)
	g := httpx.NewGroup(api, "/stubs", "Example", discardLogger())

	httpx.Get(g, "", "list-stubs", okHandler)

	op := api.OpenAPI().Paths["/stubs"].Get
	if len(op.Tags) != 1 || op.Tags[0] != "Example" {
		t.Errorf("Tags = %v, want exactly [Example]", op.Tags)
	}
	if op.OperationID != "list-stubs" {
		t.Errorf("OperationID = %q, want %q", op.OperationID, "list-stubs")
	}
	if op.Method != http.MethodGet {
		t.Errorf("Method = %q, want %q", op.Method, http.MethodGet)
	}
}

func TestGroup_ErrorsReturnsTheGroup(t *testing.T) {
	_, api := humatest.New(t)
	g := httpx.NewGroup(api, "/stubs", "Example", discardLogger())

	if got := g.Errors(httpx.Map(errGroupSentinel, http.StatusNotFound)); got != g {
		t.Errorf("Errors returned %p, want the receiver %p", got, g)
	}
}

func registeredPaths(paths map[string]*huma.PathItem) []string {
	out := make([]string, 0, len(paths))
	for p := range paths {
		out = append(out, p)
	}
	return out
}

func TestVerbs_RegisterTheirMethod(t *testing.T) {
	tests := []struct {
		name   string
		verb   func(*httpx.Group, string, string, func(*httpx.Ctx, *groupInput) (*groupOutput, error), ...httpx.Option)
		method string
		pick   func(*huma.PathItem) *huma.Operation
	}{
		{"Get", httpx.Get[groupInput, groupOutput], http.MethodGet, func(p *huma.PathItem) *huma.Operation { return p.Get }},
		{"Post", httpx.Post[groupInput, groupOutput], http.MethodPost, func(p *huma.PathItem) *huma.Operation { return p.Post }},
		{"Put", httpx.Put[groupInput, groupOutput], http.MethodPut, func(p *huma.PathItem) *huma.Operation { return p.Put }},
		{"Patch", httpx.Patch[groupInput, groupOutput], http.MethodPatch, func(p *huma.PathItem) *huma.Operation { return p.Patch }},
		{"Delete", httpx.Delete[groupInput, groupOutput], http.MethodDelete, func(p *huma.PathItem) *huma.Operation { return p.Delete }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)
			g := httpx.NewGroup(api, "/stubs", "Example", discardLogger())

			tt.verb(g, "/{id}", "probe", okHandler)

			op := tt.pick(api.OpenAPI().Paths["/stubs/{id}"])
			if op == nil {
				t.Fatalf("no %s operation registered at /stubs/{id}", tt.method)
			}
			if op.Method != tt.method {
				t.Errorf("Method = %q, want %q", op.Method, tt.method)
			}
			if len(op.Tags) != 1 || op.Tags[0] != "Example" {
				t.Errorf("Tags = %v, want exactly [Example]", op.Tags)
			}
		})
	}
}

func TestNewGroup_RejectsNilLogger(t *testing.T) {
	_, api := humatest.New(t)

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("NewGroup(nil logger) did not panic")
		}
		if msg, ok := recovered.(string); !ok || !strings.Contains(msg, "logger") {
			t.Errorf("panic = %v, want a message naming the logger", recovered)
		}
	}()

	httpx.NewGroup(api, "/stubs", "Example", nil)
}
