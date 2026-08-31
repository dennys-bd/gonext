// Command graceful is a test fixture for internal/dev's Supervisor
// tests. On start it writes MARKER_DIR/started; on SIGTERM it writes
// MARKER_DIR/graceful-stopped and exits 0, mirroring the generated
// project's own signal.NotifyContext + Echo.Shutdown pattern.
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

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM)

	_ = os.WriteFile(filepath.Join(dir, "started"), []byte(os.Getenv("MARKER_ID")), 0o644)

	select {
	case <-stop:
		_ = os.WriteFile(filepath.Join(dir, "graceful-stopped"), []byte(os.Getenv("MARKER_ID")), 0o644)
	case <-time.After(10 * time.Second):
		// Safety net so a broken test can't leak an orphan process.
	}
}
