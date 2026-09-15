package generate

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// pageProject builds a temp project root with go.mod, frontend/app/ and
// docs/openapi.yaml copied from the openapi package's fixture.
func pageProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "frontend", "app"), 0o755); err != nil {
		t.Fatalf("creating frontend/app: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("creating docs: %v", err)
	}
	data, err := os.ReadFile(filepath.Join("..", "openapi", "testdata", "openapi.yaml"))
	if err != nil {
		t.Fatalf("reading openapi fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "openapi.yaml"), data, 0o644); err != nil {
		t.Fatalf("writing docs/openapi.yaml: %v", err)
	}
	return root
}

func stubFormatFunc(t *testing.T, fn func(ctx context.Context, dir, name string, args ...string) error) {
	t.Helper()
	orig := formatFunc
	formatFunc = fn
	t.Cleanup(func() { formatFunc = orig })
}

func stubWarnOut(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	orig := warnOut
	warnOut = &buf
	t.Cleanup(func() { warnOut = orig })
	return &buf
}

func TestPage_Golden(t *testing.T) {
	tests := []struct {
		name  string
		route string
		op    string
		want  []string
	}{
		{
			name:  "get-stub",
			route: "stubs/[id]",
			op:    "get-stub",
			want:  []string{"frontend/app/stubs/[id]/page.tsx", "frontend/app/stubs/[id]/page.test.tsx"},
		},
		{
			name:  "create-stub",
			route: "stubs/new",
			op:    "create-stub",
			want: []string{
				"frontend/app/stubs/new/page.tsx",
				"frontend/app/stubs/new/actions.ts",
				"frontend/app/stubs/new/create-stub-form.tsx",
				"frontend/app/stubs/new/actions.test.ts",
				"frontend/app/stubs/new/page.test.tsx",
			},
		},
		{
			name:  "get-current-user",
			route: "me",
			op:    "get-current-user",
			want:  []string{"frontend/app/me/page.tsx", "frontend/app/me/page.test.tsx"},
		},
		{
			name:  "logout-user",
			route: "logout",
			op:    "logout-user",
			want: []string{
				"frontend/app/logout/page.tsx",
				"frontend/app/logout/actions.ts",
				"frontend/app/logout/logout-user-form.tsx",
				"frontend/app/logout/actions.test.ts",
				"frontend/app/logout/page.test.tsx",
			},
		},
		{
			name:  "update-widget",
			route: "(admin)/widgets/[...slug]/[[...rest]]",
			op:    "update-widget",
			want: []string{
				"frontend/app/(admin)/widgets/[...slug]/[[...rest]]/page.tsx",
				"frontend/app/(admin)/widgets/[...slug]/[[...rest]]/actions.ts",
				"frontend/app/(admin)/widgets/[...slug]/[[...rest]]/update-widget-form.tsx",
				"frontend/app/(admin)/widgets/[...slug]/[[...rest]]/actions.test.ts",
				"frontend/app/(admin)/widgets/[...slug]/[[...rest]]/page.test.tsx",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := pageProject(t)
			stubFormatFunc(t, func(context.Context, string, string, ...string) error { return nil })

			got, err := Page(root, tt.route, tt.op)
			if err != nil {
				t.Fatalf("Page: unexpected error: %v", err)
			}
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Fatalf("Page: paths = %v, want %v", got, tt.want)
			}

			for _, rel := range got {
				gotData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
				if err != nil {
					t.Fatalf("reading %s: %v", rel, err)
				}
				wantData, err := os.ReadFile(filepath.Join("testdata", "page", tt.name, filepath.Base(rel)))
				if err != nil {
					t.Fatalf("reading golden %s: %v", rel, err)
				}
				if !bytes.Equal(gotData, wantData) {
					t.Errorf("%s: content differs\n--- want\n%s\n--- got\n%s", rel, wantData, gotData)
				}
			}

			// Assert no extra files were written under the route directory.
			routeDir := filepath.Dir(filepath.Join(root, filepath.FromSlash(got[0])))
			entries, err := os.ReadDir(routeDir)
			if err != nil {
				t.Fatalf("reading route dir: %v", err)
			}
			gotNames := make([]string, 0, len(entries))
			for _, e := range entries {
				gotNames = append(gotNames, e.Name())
			}
			wantNames := make([]string, 0, len(tt.want))
			for _, w := range tt.want {
				wantNames = append(wantNames, filepath.Base(w))
			}
			sort.Strings(gotNames)
			sort.Strings(wantNames)
			if strings.Join(gotNames, ",") != strings.Join(wantNames, ",") {
				t.Errorf("route dir entries = %v, want %v", gotNames, wantNames)
			}
		})
	}
}

func TestPage_Errors(t *testing.T) {
	tests := []struct {
		name    string
		route   string
		op      string
		wantErr string
	}{
		{
			name:    "missing document",
			route:   "stubs/[id]",
			op:      "get-stub",
			wantErr: "docs/openapi.yaml not found; run `gonext openapi`",
		},
		{
			name:    "unknown operation",
			route:   "stubs/[id]",
			op:      "get-stubb",
			wantErr: `no operation "get-stubb" in docs/openapi.yaml`,
		},
		{
			name:    "unsupported method",
			route:   "healthz",
			op:      "head-healthz",
			wantErr: `unsupported method HEAD for operation "head-healthz"`,
		},
		{
			name:    "invalid route",
			route:   "_x/y",
			op:      "get-stub",
			wantErr: `invalid route "_x/y": Next.js does not route pages under a private folder`,
		},
		{
			name:    "path param not satisfiable",
			route:   "stubs/new",
			op:      "get-stub",
			wantErr: `operation "get-stub" needs path parameter "id": add an [id] segment to the route`,
		},
		{
			name:    "optional catch-all cannot satisfy",
			route:   "stubs/[[...id]]",
			op:      "get-stub",
			wantErr: `operation "get-stub" needs path parameter "id": add an [id] segment to the route`,
		},
		{
			name:    "required query parameter",
			route:   "widgets",
			op:      "list-widgets",
			wantErr: `unsupported: operation "list-widgets" has a required query parameter "page"`,
		},
		{
			name:    "no application/json body",
			route:   "widgets/[slug]",
			op:      "upload-widget",
			wantErr: `unsupported: operation "upload-widget" has no application/json request body`,
		},
		{
			name:    "unresolvable $ref",
			route:   "widgets/[slug]",
			op:      "rename-widget",
			wantErr: `unsupported $ref "#/definitions/RenameWidgetInputBody" in docs/openapi.yaml`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n\ngo 1.26\n"), 0o644); err != nil {
				t.Fatalf("writing go.mod: %v", err)
			}
			if err := os.MkdirAll(filepath.Join(root, "frontend", "app"), 0o755); err != nil {
				t.Fatalf("creating frontend/app: %v", err)
			}
			if tt.name != "missing document" {
				if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
					t.Fatalf("creating docs: %v", err)
				}
				data, err := os.ReadFile(filepath.Join("..", "openapi", "testdata", "openapi.yaml"))
				if err != nil {
					t.Fatalf("reading fixture: %v", err)
				}
				if err := os.WriteFile(filepath.Join(root, "docs", "openapi.yaml"), data, 0o644); err != nil {
					t.Fatalf("writing fixture: %v", err)
				}
			}

			_, err := Page(root, tt.route, tt.op)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Page: err = %v, want %q", err, tt.wantErr)
			}
			if _, statErr := os.Stat(filepath.Join(root, "frontend", "app", tt.route)); !os.IsNotExist(statErr) {
				t.Errorf("Page: expected frontend/app/%s not to exist, stat err = %v", tt.route, statErr)
			}
		})
	}
}

func TestPage_RefusesExistingTarget(t *testing.T) {
	root := pageProject(t)
	dir := filepath.Join(root, "frontend", "app", "stubs", "new")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "actions.ts"), []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("pre-creating actions.ts: %v", err)
	}

	_, err := Page(root, "stubs/new", "create-stub")
	wantErr := "frontend/app/stubs/new/actions.ts already exists"
	if err == nil || err.Error() != wantErr {
		t.Fatalf("Page: err = %v, want %q", err, wantErr)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "page.tsx")); !os.IsNotExist(statErr) {
		t.Errorf("Page: expected page.tsx (earlier in write order) not to have been created, stat err = %v", statErr)
	}
}

func TestPage_FormatsWrittenFiles(t *testing.T) {
	root := pageProject(t)

	var gotDir string
	var gotArgv []string
	stubFormatFunc(t, func(_ context.Context, dir, name string, args ...string) error {
		gotDir = dir
		gotArgv = append([]string{name}, args...)
		return nil
	})

	if _, err := Page(root, "stubs/[id]", "get-stub"); err != nil {
		t.Fatalf("Page: unexpected error: %v", err)
	}

	if gotDir != filepath.Join(root, "frontend") {
		t.Errorf("formatFunc dir = %q, want %q", gotDir, filepath.Join(root, "frontend"))
	}
	want := `pnpm exec biome check --write app/stubs/[id]/page.tsx app/stubs/[id]/page.test.tsx`
	if strings.Join(gotArgv, " ") != want {
		t.Errorf("formatFunc argv = %q, want %q", strings.Join(gotArgv, " "), want)
	}
}

func TestPage_FormatFailureIsWarning(t *testing.T) {
	root := pageProject(t)
	stubFormatFunc(t, func(context.Context, string, string, ...string) error {
		return errors.New("pnpm: command not found")
	})
	buf := stubWarnOut(t)

	paths, err := Page(root, "stubs/[id]", "get-stub")
	if err != nil {
		t.Fatalf("Page: unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "pnpm format") {
		t.Errorf("warnOut = %q, want it to mention `pnpm format`", buf.String())
	}
	for _, rel := range paths {
		if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); statErr != nil {
			t.Errorf("Page: %s was not written despite the formatting failure: %v", rel, statErr)
		}
	}
}

// TestPage_UnusedRouteParamOmitsParamsProp covers the MEDIUM finding:
// a route declaring params an operation never uses must not leave an
// unused `params` prop on the page.
func TestPage_UnusedRouteParamOmitsParamsProp(t *testing.T) {
	root := pageProject(t)
	stubFormatFunc(t, func(context.Context, string, string, ...string) error { return nil })

	got, err := Page(root, "things/[id]", "get-current-user")
	if err != nil {
		t.Fatalf("Page: unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(got[0])))
	if err != nil {
		t.Fatalf("reading %s: %v", got[0], err)
	}
	if strings.Contains(string(data), "params") {
		t.Errorf("page.tsx contains %q, want no params prop/type/await for an operation that uses no route params:\n%s", "params", data)
	}
}

// TestPage_TargetStatErrorSurfaces covers the MEDIUM finding: a stat
// error other than "not exist" (here ENOTDIR, since the route
// directory itself is a file) must be reported, not swallowed as
// "absent".
func TestPage_TargetStatErrorSurfaces(t *testing.T) {
	root := pageProject(t)
	if err := os.MkdirAll(filepath.Join(root, "frontend", "app", "stubs"), 0o755); err != nil {
		t.Fatalf("creating frontend/app/stubs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "frontend", "app", "stubs", "new"), []byte("not a directory\n"), 0o644); err != nil {
		t.Fatalf("pre-creating stubs/new as a file: %v", err)
	}

	_, err := Page(root, "stubs/new", "create-stub")
	if err == nil {
		t.Fatal("Page: expected an error")
	}
	if strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Page: err = %v, want a stat error surfaced, not treated as already-exists", err)
	}
	if !strings.Contains(err.Error(), "checking") {
		t.Errorf("Page: err = %v, want it to wrap the stat failure", err)
	}
}

// TestPage_RejectsHostilePropertyName covers the CRITICAL finding: a
// request body property whose name is not a valid JS identifier, and
// a summary containing a double quote, must not reach any rendered
// file verbatim.
func TestPage_RejectsHostilePropertyName(t *testing.T) {
	root := pageProject(t)
	stubFormatFunc(t, func(context.Context, string, string, ...string) error { return nil })
	buf := stubWarnOut(t)

	got, err := Page(root, "test-injection", "test-injection")
	if err != nil {
		t.Fatalf("Page: unexpected error: %v", err)
	}

	for _, rel := range got {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		if strings.Contains(string(data), "bad-name") {
			t.Errorf("%s contains the raw hostile property name %q:\n%s", rel, "bad-name", data)
		}
		if strings.Contains(string(data), `Test "injection" safety`) {
			t.Errorf("%s contains the raw unsafe summary:\n%s", rel, data)
		}
	}

	if !strings.Contains(buf.String(), `skipped property "bad-name": not a valid identifier`) {
		t.Errorf("warnOut = %q, want it to contain a warning for the hostile property name", buf.String())
	}
}

func TestPage_WarnsOnSkippedProperties(t *testing.T) {
	root := pageProject(t)
	stubFormatFunc(t, func(context.Context, string, string, ...string) error { return nil })
	buf := stubWarnOut(t)

	if _, err := Page(root, "widgets/[slug]", "update-widget"); err != nil {
		t.Fatalf("Page: unexpected error: %v", err)
	}
	for _, want := range []string{`skipped non-primitive property "meta"`, `skipped non-primitive property "tags"`} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("warnOut = %q, want it to contain %q", buf.String(), want)
		}
	}
}
