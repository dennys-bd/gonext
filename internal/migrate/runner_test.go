package migrate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApply_SubstitutesModulePathAndCleansUpOnSuccess(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/foo")
	backendDir := mkBackendDir(t, root)

	var capturedContent string
	defer stubRunFunc(t, func(ctx context.Context, dir, name string, args ...string) error {
		data, err := os.ReadFile(filepath.Join(dir, runnerFilename))
		if err != nil {
			t.Fatalf("reading runner file during run: %v", err)
		}
		capturedContent = string(data)
		return nil
	})()

	if err := Apply(context.Background(), root); err != nil {
		t.Fatalf("Apply: unexpected error: %v", err)
	}

	if !strings.Contains(capturedContent, "example.com/foo/backend/internal/config") {
		t.Errorf("Apply: runner content missing substituted module path, got: %s", capturedContent)
	}
	if strings.Contains(capturedContent, modulePathToken) {
		t.Errorf("Apply: runner content still contains unsubstituted token %q", modulePathToken)
	}

	if _, err := os.Stat(filepath.Join(backendDir, runnerFilename)); !os.IsNotExist(err) {
		t.Errorf("Apply: expected runner file to be removed, stat err = %v", err)
	}
}

func TestApply_CleansUpRunnerOnRunError(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/foo")
	backendDir := mkBackendDir(t, root)

	wantErr := errors.New("boom")
	defer stubRunFunc(t, func(ctx context.Context, dir, name string, args ...string) error {
		return wantErr
	})()

	err := Apply(context.Background(), root)
	if err == nil {
		t.Fatalf("Apply: expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("Apply: expected wrapped %v, got %v", wantErr, err)
	}

	if _, statErr := os.Stat(filepath.Join(backendDir, runnerFilename)); !os.IsNotExist(statErr) {
		t.Errorf("Apply: expected runner file to be removed after run error, stat err = %v", statErr)
	}
}

func TestApply_ErrorsWhenModuleNotFound(t *testing.T) {
	root := t.TempDir() // no go.mod

	if err := Apply(context.Background(), root); err == nil {
		t.Errorf("Apply: expected error when go.mod missing, got nil")
	}
}

func TestApply_UsesGoRunFromBackendDirWithRunnerFilename(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/foo")
	backendDir := mkBackendDir(t, root)

	var gotDir, gotName string
	var gotArgs []string
	defer stubRunFunc(t, func(ctx context.Context, dir, name string, args ...string) error {
		gotDir, gotName, gotArgs = dir, name, args
		return nil
	})()

	if err := Apply(context.Background(), root); err != nil {
		t.Fatalf("Apply: unexpected error: %v", err)
	}

	if gotDir != backendDir {
		t.Errorf("Apply: dir = %q, want %q (the runner must live under backend/ to see its internal/ packages)", gotDir, backendDir)
	}
	if gotName != "go" {
		t.Errorf("Apply: name = %q, want %q", gotName, "go")
	}
	if len(gotArgs) != 2 || gotArgs[0] != "run" || gotArgs[1] != runnerFilename {
		t.Errorf("Apply: args = %v, want [run %s]", gotArgs, runnerFilename)
	}
}

func mkBackendDir(t *testing.T, root string) string {
	t.Helper()
	dir := filepath.Join(root, "backend")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating backend dir: %v", err)
	}
	return dir
}

func stubRunFunc(t *testing.T, fn func(ctx context.Context, dir, name string, args ...string) error) func() {
	t.Helper()
	orig := runFunc
	runFunc = fn
	return func() { runFunc = orig }
}
