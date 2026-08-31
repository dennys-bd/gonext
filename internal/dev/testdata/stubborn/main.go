// Command stubborn is a test fixture for internal/dev's Supervisor
// tests. It ignores SIGTERM so tests can exercise the Supervisor's
// SIGKILL fallback path.
package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	dir := os.Getenv("MARKER_DIR")
	signal.Ignore(syscall.SIGTERM)

	_ = os.WriteFile(filepath.Join(dir, "started"), []byte(os.Getenv("MARKER_ID")), 0o644)

	// Bounds the process lifetime in case a test fails to kill it;
	// the Supervisor's SIGKILL always arrives well before this.
	time.Sleep(10 * time.Second)
}
