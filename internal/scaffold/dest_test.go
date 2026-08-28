package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDest_DefaultsToName(t *testing.T) {
	got, err := ResolveDest("my-app", "")
	if err != nil {
		t.Fatalf("ResolveDest: unexpected error: %v", err)
	}
	want := filepath.Join(".", "my-app")
	if got != want {
		t.Errorf("ResolveDest(%q, \"\") = %q, want %q", "my-app", got, want)
	}
}

func TestResolveDest_UsesGivenPath(t *testing.T) {
	got, err := ResolveDest("my-app", "/tmp/somewhere")
	if err != nil {
		t.Fatalf("ResolveDest: unexpected error: %v", err)
	}
	if got != "/tmp/somewhere" {
		t.Errorf("ResolveDest(%q, %q) = %q, want %q", "my-app", "/tmp/somewhere", got, "/tmp/somewhere")
	}
}

func TestCheckEmpty_MissingDirIsFine(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	if err := CheckEmpty(dir); err != nil {
		t.Errorf("CheckEmpty(%q): unexpected error: %v", dir, err)
	}
}

func TestCheckEmpty_EmptyDirIsFine(t *testing.T) {
	dir := t.TempDir()
	if err := CheckEmpty(dir); err != nil {
		t.Errorf("CheckEmpty(%q): unexpected error: %v", dir, err)
	}
}

func TestCheckEmpty_NonEmptyDirErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("data"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := CheckEmpty(dir); err == nil {
		t.Errorf("CheckEmpty(%q): expected error for non-empty dir, got nil", dir)
	}
}
