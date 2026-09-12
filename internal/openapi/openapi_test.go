package openapi

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dennys-bd/gonext/internal/project"
)

func TestGenerate_SubstitutesModulePathAndCleansUp(t *testing.T) {
	root := newProject(t, "example.com/foo")

	var capturedContent string
	stubRun(t, func(_ context.Context, dir, _ string, args ...string) ([]byte, error) {
		data, err := os.ReadFile(filepath.Join(dir, runnerFilename))
		if err != nil {
			t.Fatalf("reading runner during run: %v", err)
		}
		capturedContent = string(data)
		if len(args) != 2 || args[0] != "run" || args[1] != runnerFilename {
			t.Errorf("Generate: args = %v, want [run %s]", args, runnerFilename)
		}
		return []byte("openapi: 3.1.0\n"), nil
	})

	doc, err := Generate(context.Background(), root)
	if err != nil {
		t.Fatalf("Generate: unexpected error: %v", err)
	}
	if string(doc) != "openapi: 3.1.0\n" {
		t.Errorf("Generate: doc = %q, want the runner's stdout", doc)
	}
	if !strings.Contains(capturedContent, "example.com/foo/backend/internal/openapi") || strings.Contains(capturedContent, project.ModulePathToken) {
		t.Errorf("Generate: module path not substituted in runner: %s", capturedContent)
	}
	if _, err := os.Stat(filepath.Join(root, "backend", runnerFilename)); !os.IsNotExist(err) {
		t.Errorf("Generate: expected runner removed, stat err = %v", err)
	}
}

func TestGenerate_CleansUpOnRunError(t *testing.T) {
	root := newProject(t, "example.com/foo")
	wantErr := errors.New("boom")
	stubRun(t, func(context.Context, string, string, ...string) ([]byte, error) { return nil, wantErr })

	if _, err := Generate(context.Background(), root); !errors.Is(err, wantErr) {
		t.Errorf("Generate: err = %v, want wrapped %v", err, wantErr)
	}
	if _, err := os.Stat(filepath.Join(root, "backend", runnerFilename)); !os.IsNotExist(err) {
		t.Errorf("Generate: expected runner removed after error, stat err = %v", err)
	}
}

func TestWrite_WritesDocumentThenRunsCodegen(t *testing.T) {
	root := newProject(t, "example.com/foo")
	stubRun(t, func(context.Context, string, string, ...string) ([]byte, error) { return []byte("doc\n"), nil })
	var codegenDir string
	var codegenArgs []string
	stubCodegen(t, func(_ context.Context, dir, name string, args ...string) error {
		codegenDir, codegenArgs = dir, append([]string{name}, args...)
		return nil
	})

	if err := Write(context.Background(), root); err != nil {
		t.Fatalf("Write: unexpected error: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, DocumentPath))
	if err != nil {
		t.Fatalf("reading written document: %v", err)
	}
	if string(got) != "doc\n" {
		t.Errorf("Write: document = %q, want %q", got, "doc\n")
	}
	if codegenDir != filepath.Join(root, "frontend") || strings.Join(codegenArgs, " ") != "pnpm codegen" {
		t.Errorf("Write: codegen = %v in %q, want `pnpm codegen` in frontend/", codegenArgs, codegenDir)
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name      string
		committed string
		generated string
		wantErr   error
	}{
		{name: "matches", committed: "doc\n", generated: "doc\n"},
		{name: "stale", committed: "old\n", generated: "new\n", wantErr: ErrStale},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := newProject(t, "example.com/foo")
			if err := os.WriteFile(filepath.Join(root, DocumentPath), []byte(tt.committed), 0o644); err != nil {
				t.Fatalf("writing committed document: %v", err)
			}
			stubRun(t, func(context.Context, string, string, ...string) ([]byte, error) { return []byte(tt.generated), nil })

			if err := Check(context.Background(), root); !errors.Is(err, tt.wantErr) {
				t.Errorf("Check: err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheck_ErrorsWhenDocumentMissing(t *testing.T) {
	root := newProject(t, "example.com/foo")
	stubRun(t, func(context.Context, string, string, ...string) ([]byte, error) { return []byte("doc\n"), nil })

	if err := Check(context.Background(), root); err == nil || errors.Is(err, ErrStale) {
		t.Errorf("Check: err = %v, want a read error distinct from ErrStale", err)
	}
}

// newProject lays out the minimal project shape the package needs:
// a go.mod, backend/ and docs/.
func newProject(t *testing.T, modulePath string) string {
	t.Helper()
	root := t.TempDir()
	content := "module " + modulePath + "\n\ngo 1.26\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}
	for _, dir := range []string{"backend", "docs", "frontend"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("creating %s: %v", dir, err)
		}
	}
	return root
}

func stubRun(t *testing.T, fn func(ctx context.Context, dir, name string, args ...string) ([]byte, error)) {
	t.Helper()
	orig := runFunc
	runFunc = fn
	t.Cleanup(func() { runFunc = orig })
}

func stubCodegen(t *testing.T, fn func(ctx context.Context, dir, name string, args ...string) error) {
	t.Helper()
	orig := codegenFunc
	codegenFunc = fn
	t.Cleanup(func() { codegenFunc = orig })
}
