package dev

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBuild_Success(t *testing.T) {
	root := newFixtureModule(t, "package main\n\nfunc main() {}\n")

	if err := Build(context.Background(), root); err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, BinaryPath)); err != nil {
		t.Errorf("expected binary at %s: %v", BinaryPath, err)
	}
}

func TestBuild_FailureLeavesExistingBinaryUntouched(t *testing.T) {
	root := newFixtureModule(t, "package main\n\nfunc main() {}\n")

	if err := Build(context.Background(), root); err != nil {
		t.Fatalf("initial Build: unexpected error: %v", err)
	}
	before, err := os.ReadFile(filepath.Join(root, BinaryPath))
	if err != nil {
		t.Fatalf("reading initial binary: %v", err)
	}

	// Introduce a compile error.
	mainGo := filepath.Join(root, "backend", "main.go")
	if err := os.WriteFile(mainGo, []byte("package main\n\nfunc main() { this is not valid go }\n"), 0o644); err != nil {
		t.Fatalf("writing broken main.go: %v", err)
	}

	if err := Build(context.Background(), root); err == nil {
		t.Fatalf("Build: expected error for invalid source, got nil")
	}

	after, err := os.ReadFile(filepath.Join(root, BinaryPath))
	if err != nil {
		t.Fatalf("reading binary after failed build: %v", err)
	}
	if string(before) != string(after) {
		t.Errorf("expected binary to be untouched after a failed build")
	}
}

// newFixtureModule creates a t.TempDir() containing a minimal Go
// module with backend/main.go set to src, suitable for exercising
// Build against a real go build invocation.
func newFixtureModule(t *testing.T, src string) string {
	t.Helper()
	root := t.TempDir()

	goMod := "module fixture.test/app\n\ngo 1.26\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	backendDir := filepath.Join(root, "backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("creating backend dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("writing backend/main.go: %v", err)
	}
	return root
}
