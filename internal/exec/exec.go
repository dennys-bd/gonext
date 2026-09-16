// Package exec wraps os/exec for the scaffolding CLI's long-running,
// streamed subprocess steps (go mod tidy, go get -tool, pnpm install).
package exec

import (
	"context"
	"io"
	"os"
	"os/exec"
)

// Run executes name with args in dir, streaming stdout/stderr live to the
// parent process. stdin is not forwarded, so a tool that would prompt
// (pnpm, docker) gets EOF instead of blocking on the terminal.
func Run(ctx context.Context, dir string, name string, args ...string) error {
	return run(ctx, dir, nil, name, args...)
}

// RunInteractive is Run with the parent's stdin forwarded, for the
// one step that legitimately asks the developer something: the
// migration runner's rollback confirmation.
func RunInteractive(ctx context.Context, dir string, name string, args ...string) error {
	return run(ctx, dir, os.Stdin, name, args...)
}

func run(ctx context.Context, dir string, stdin io.Reader, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
