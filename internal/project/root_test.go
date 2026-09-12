package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoot_AcceptsDirWithGoMod(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "example.com/foo")

	root, err := Root(dir)
	if err != nil {
		t.Fatalf("Root: unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("Root: got %q, want %q", root, dir)
	}
}

// TestRoot_RejectsSubdirectory pins the decision that subcommands run
// from the project root: a go.mod in an ancestor is not enough.
func TestRoot_RejectsSubdirectory(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/foo")
	nested := filepath.Join(root, "backend", "users")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("creating nested dir: %v", err)
	}

	if _, err := Root(nested); err == nil {
		t.Errorf("Root: expected error from a subdirectory, got nil")
	}
}

func TestRoot_RejectsDirWithoutGoMod(t *testing.T) {
	if _, err := Root(t.TempDir()); err == nil {
		t.Errorf("Root: expected error when no go.mod present, got nil")
	}
}

func TestModulePath_ParsesModuleLine(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "example.com/foo")

	got, err := ModulePath(dir)
	if err != nil {
		t.Fatalf("ModulePath: unexpected error: %v", err)
	}
	if got != "example.com/foo" {
		t.Errorf("ModulePath: got %q, want %q", got, "example.com/foo")
	}
}

func TestModulePath_ErrorsOnMissingGoMod(t *testing.T) {
	dir := t.TempDir()
	if _, err := ModulePath(dir); err == nil {
		t.Errorf("ModulePath: expected error when go.mod missing, got nil")
	}
}

func writeGoMod(t *testing.T, dir, modulePath string) {
	t.Helper()
	content := "module " + modulePath + "\n\ngo 1.26\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}
}
