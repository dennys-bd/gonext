package exec

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRun_Success(t *testing.T) {
	dir := t.TempDir()
	if err := Run(context.Background(), dir, "true"); err != nil {
		t.Errorf("Run(true): unexpected error: %v", err)
	}
}

func TestRun_Failure(t *testing.T) {
	dir := t.TempDir()
	if err := Run(context.Background(), dir, "false"); err == nil {
		t.Errorf("Run(false): expected error, got nil")
	}
}

func TestRun_UsesDir(t *testing.T) {
	dir := t.TempDir()
	if err := Run(context.Background(), dir, "touch", "marker.txt"); err != nil {
		t.Fatalf("Run(touch): unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "marker.txt")); err != nil {
		t.Errorf("expected marker.txt to be created in %q: %v", dir, err)
	}
}

// TestRunInteractive_ForwardsStdin swaps the package-level os.Stdin
// for a pipe, so it must not run in parallel with other tests here.
func TestRunInteractive_ForwardsStdin(t *testing.T) {
	dir := t.TempDir()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	if _, err := w.WriteString("y\n"); err != nil {
		t.Fatalf("writing to pipe: %v", err)
	}
	w.Close()

	if err := RunInteractive(context.Background(), dir, "sh", "-c", `read x && test "$x" = y`); err != nil {
		t.Errorf("RunInteractive: expected the subprocess to read %q from the forwarded stdin, got error: %v", "y", err)
	}
}

func TestRun_DoesNotForwardStdin(t *testing.T) {
	dir := t.TempDir()
	// With stdin at /dev/null, read hits EOF and fails; the test
	// passes only because Run does not hand the subprocess a stdin.
	if err := Run(context.Background(), dir, "sh", "-c", `read x`); err == nil {
		t.Error("Run: expected read from an unforwarded stdin to fail, got nil")
	}
}
