package dev

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestSupervisor_StartWritesStartedMarker(t *testing.T) {
	bin := buildFixture(t, "graceful")
	dir := t.TempDir()
	env := append(os.Environ(), "MARKER_DIR="+dir, "MARKER_ID=1")

	s := NewSupervisor(WithStopTimeout(500 * time.Millisecond))
	if err := s.Start(bin, env); err != nil {
		t.Fatalf("Start: unexpected error: %v", err)
	}
	defer s.Stop()

	waitForFile(t, filepath.Join(dir, "started"))
}

func TestSupervisor_RestartGracefullyStopsPrevious(t *testing.T) {
	bin := buildFixture(t, "graceful")
	dir := t.TempDir()

	s := NewSupervisor(WithStopTimeout(2 * time.Second))

	env1 := append(os.Environ(), "MARKER_DIR="+dir, "MARKER_ID=first")
	if err := s.Start(bin, env1); err != nil {
		t.Fatalf("first Start: unexpected error: %v", err)
	}
	waitForFile(t, filepath.Join(dir, "started"))
	os.Remove(filepath.Join(dir, "started"))

	env2 := append(os.Environ(), "MARKER_DIR="+dir, "MARKER_ID=second")
	if err := s.Start(bin, env2); err != nil {
		t.Fatalf("second Start: unexpected error: %v", err)
	}
	defer s.Stop()

	waitForFileContent(t, filepath.Join(dir, "graceful-stopped"), "first")
	waitForFileContent(t, filepath.Join(dir, "started"), "second")
}

func TestSupervisor_StopFallsBackToSIGKILLWhenChildIgnoresSIGTERM(t *testing.T) {
	bin := buildFixture(t, "stubborn")
	dir := t.TempDir()
	env := append(os.Environ(), "MARKER_DIR="+dir, "MARKER_ID=1")

	s := NewSupervisor(WithStopTimeout(100 * time.Millisecond))
	if err := s.Start(bin, env); err != nil {
		t.Fatalf("Start: unexpected error: %v", err)
	}
	waitForFile(t, filepath.Join(dir, "started"))

	stopped := make(chan error, 1)
	go func() { stopped <- s.Stop() }()

	select {
	case err := <-stopped:
		if err != nil {
			t.Errorf("Stop: unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Stop: expected SIGKILL fallback to terminate the stubborn child within 2s")
	}
}

// buildFixture compiles internal/dev/testdata/<name> into a fresh
// binary in t.TempDir() and returns its path.
func buildFixture(t *testing.T, name string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), name)
	cmd := exec.CommandContext(context.Background(), "go", "build", "-o", out, "./testdata/"+name)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("building fixture %s: %v", name, err)
	}
	return out
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s to exist", path)
}

func waitForFileContent(t *testing.T, path, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			if string(data) != want {
				t.Fatalf("%s content = %q, want %q", path, data, want)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s to exist", path)
}
