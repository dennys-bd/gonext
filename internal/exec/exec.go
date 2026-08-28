// Package exec wraps os/exec for the scaffolding CLI's long-running,
// streamed subprocess steps (go mod tidy, pnpm install, docker
// compose, the generated project's migrate binary).
package exec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// DefaultHealthTimeout is the bounded timeout used when polling a
// service (e.g. Postgres via pg_isready) for health.
const DefaultHealthTimeout = 30 * time.Second

// Run executes name with args in dir, streaming stdout/stderr live
// to the parent process so long-running steps show progress.
func Run(ctx context.Context, dir string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// WaitHealthy polls checkFn at a short interval until it succeeds or
// timeout elapses, returning an error in the latter case.
func WaitHealthy(ctx context.Context, timeout time.Duration, checkFn func(context.Context) error) error {
	deadline := time.Now().Add(timeout)

	for {
		if err := checkFn(ctx); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for health check", timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}
