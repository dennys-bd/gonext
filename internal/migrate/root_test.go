package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRoot_FindsGoModInStartDir(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "example.com/foo")

	root, err := ResolveRoot(dir)
	if err != nil {
		t.Fatalf("ResolveRoot: unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("ResolveRoot: got %q, want %q", root, dir)
	}
}

func TestResolveRoot_FindsGoModInAncestor(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/foo")

	nested := filepath.Join(root, "backend", "cmd", "server")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	got, err := ResolveRoot(nested)
	if err != nil {
		t.Fatalf("ResolveRoot: unexpected error: %v", err)
	}
	if got != root {
		t.Errorf("ResolveRoot: got %q, want %q", got, root)
	}
}

func TestResolveRoot_NotFound(t *testing.T) {
	dir := t.TempDir()
	if _, err := ResolveRoot(dir); err == nil {
		t.Errorf("ResolveRoot: expected error when no go.mod present, got nil")
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
