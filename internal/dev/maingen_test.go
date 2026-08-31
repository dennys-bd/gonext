package dev

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestRegenerateMain_SubstitutesModulePath(t *testing.T) {
	fsys := fstest.MapFS{
		mainGoTemplatePath: &fstest.MapFile{
			Data: []byte("package main\n\nimport \"[PROJECT-NAME]/backend/example\"\n"),
		},
	}
	root := t.TempDir()

	if err := RegenerateMain(fsys, root, "github.com/acme/widgets"); err != nil {
		t.Fatalf("RegenerateMain: unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "backend", "main.go"))
	if err != nil {
		t.Fatalf("reading generated main.go: %v", err)
	}
	want := "package main\n\nimport \"github.com/acme/widgets/backend/example\"\n"
	if string(got) != want {
		t.Errorf("main.go = %q, want %q", got, want)
	}
}

func TestRegenerateMain_OverwritesExistingContent(t *testing.T) {
	fsys := fstest.MapFS{
		mainGoTemplatePath: &fstest.MapFile{
			Data: []byte("package main\n// canonical\n"),
		},
	}
	root := t.TempDir()
	backendDir := filepath.Join(root, "backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	existing := filepath.Join(backendDir, "main.go")
	if err := os.WriteFile(existing, []byte("package main\n// hand-edited, should be clobbered\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := RegenerateMain(fsys, root, "example.com/app"); err != nil {
		t.Fatalf("RegenerateMain: unexpected error: %v", err)
	}

	got, err := os.ReadFile(existing)
	if err != nil {
		t.Fatalf("reading generated main.go: %v", err)
	}
	want := "package main\n// canonical\n"
	if string(got) != want {
		t.Errorf("main.go = %q, want %q (hand-edit should be unconditionally overwritten)", got, want)
	}
}

func TestRegenerateMain_ErrorsWhenTemplateMissing(t *testing.T) {
	fsys := fstest.MapFS{}
	root := t.TempDir()

	if err := RegenerateMain(fsys, root, "example.com/app"); err == nil {
		t.Errorf("RegenerateMain: expected error when template file is missing, got nil")
	}
}
