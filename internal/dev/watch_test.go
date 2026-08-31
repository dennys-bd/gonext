package dev

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShouldTrigger_GoFileUnderBackend(t *testing.T) {
	backendRoot := filepath.FromSlash("/tmp/proj/backend")
	path := filepath.Join(backendRoot, "app.go")
	if !shouldTrigger(backendRoot, path) {
		t.Errorf("shouldTrigger(%q) = false, want true", path)
	}
}

func TestShouldTrigger_IgnoresNonGoFiles(t *testing.T) {
	backendRoot := filepath.FromSlash("/tmp/proj/backend")
	path := filepath.Join(backendRoot, "README.md")
	if shouldTrigger(backendRoot, path) {
		t.Errorf("shouldTrigger(%q) = true, want false", path)
	}
}

func TestShouldTrigger_IgnoresBinDirectory(t *testing.T) {
	backendRoot := filepath.FromSlash("/tmp/proj/backend")
	path := filepath.Join(backendRoot, "bin", "dev-server.go")
	if shouldTrigger(backendRoot, path) {
		t.Errorf("shouldTrigger(%q) = true, want false (backend/bin/ must be excluded)", path)
	}
}

func TestShouldTrigger_GoFileInNestedPackage(t *testing.T) {
	backendRoot := filepath.FromSlash("/tmp/proj/backend")
	path := filepath.Join(backendRoot, "users", "internal", "application", "register.go")
	if !shouldTrigger(backendRoot, path) {
		t.Errorf("shouldTrigger(%q) = false, want true", path)
	}
}

func TestDebouncer_CollapsesBurstIntoOneSignal(t *testing.T) {
	d := NewDebouncer(20 * time.Millisecond)
	defer d.Stop()

	for range 5 {
		d.Trigger()
		time.Sleep(2 * time.Millisecond)
	}

	select {
	case <-d.C:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected a debounced signal within 200ms, got none")
	}

	select {
	case <-d.C:
		t.Fatalf("expected only one debounced signal for the burst, got a second")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestDebouncer_FiresAgainAfterQuietPeriod(t *testing.T) {
	d := NewDebouncer(20 * time.Millisecond)
	defer d.Stop()

	d.Trigger()
	select {
	case <-d.C:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected first debounced signal within 200ms, got none")
	}

	d.Trigger()
	select {
	case <-d.C:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected second debounced signal within 200ms, got none")
	}
}

func TestWatch_TriggersOnGoFileChange(t *testing.T) {
	backendRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(backendRoot, "users"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	trigger := make(chan struct{}, 8)
	stop, err := Watch(backendRoot, 20*time.Millisecond, func() { trigger <- struct{}{} })
	if err != nil {
		t.Fatalf("Watch: unexpected error: %v", err)
	}
	defer stop()

	// Give the watcher time to finish registering directories before
	// the write below, matching its own startup ordering.
	time.Sleep(100 * time.Millisecond)

	target := filepath.Join(backendRoot, "users", "users.go")
	if err := os.WriteFile(target, []byte("package users\n"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", target, err)
	}

	select {
	case <-trigger:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected a rebuild trigger after writing a .go file, got none")
	}
}

func TestWatch_IgnoresNonGoFileChange(t *testing.T) {
	backendRoot := t.TempDir()

	trigger := make(chan struct{}, 8)
	stop, err := Watch(backendRoot, 20*time.Millisecond, func() { trigger <- struct{}{} })
	if err != nil {
		t.Fatalf("Watch: unexpected error: %v", err)
	}
	defer stop()

	time.Sleep(100 * time.Millisecond)

	target := filepath.Join(backendRoot, "notes.txt")
	if err := os.WriteFile(target, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", target, err)
	}

	select {
	case <-trigger:
		t.Fatalf("expected no rebuild trigger for a non-.go file")
	case <-time.After(300 * time.Millisecond):
	}
}

func TestWatch_RegistersNewSubdirectories(t *testing.T) {
	backendRoot := t.TempDir()

	trigger := make(chan struct{}, 8)
	stop, err := Watch(backendRoot, 20*time.Millisecond, func() { trigger <- struct{}{} })
	if err != nil {
		t.Fatalf("Watch: unexpected error: %v", err)
	}
	defer stop()

	time.Sleep(100 * time.Millisecond)

	newDir := filepath.Join(backendRoot, "newpkg")
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", newDir, err)
	}
	// Give the create-event handler time to register the new directory.
	time.Sleep(150 * time.Millisecond)

	target := filepath.Join(newDir, "newpkg.go")
	if err := os.WriteFile(target, []byte("package newpkg\n"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", target, err)
	}

	select {
	case <-trigger:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected a rebuild trigger for a .go file in a newly created subdirectory")
	}
}
