package dev

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/dennys-bd/gonext/internal/migrate"
)

// debounceWindow is the fixed burst-collapsing window for the
// watcher; per the design's non-goals this is not configurable.
const debounceWindow = 200 * time.Millisecond

// runLoop wires the watcher's debounced trigger to a rebuild+restart
// step. rebuild/restart/stop are injected so the loop's properties —
// a failed rebuild never touches the running process, a successful
// one restarts it, cancellation stops it — are testable without the
// real embedded template or a `go build` toolchain invocation.
type runLoop struct {
	rebuild func(ctx context.Context) error
	restart func() error
	stop    func() error
}

// onChange runs one rebuild+restart cycle. A failed rebuild is
// reported and otherwise ignored — the currently running process, if
// any, is left untouched.
func (l *runLoop) onChange(ctx context.Context) {
	if err := l.rebuild(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "gonext dev: build failed:", err)
		return
	}
	if err := l.restart(); err != nil {
		fmt.Fprintln(os.Stderr, "gonext dev: restart failed:", err)
	}
}

// run performs an initial rebuild+restart, then watches backendRoot
// until ctx is cancelled, rebuilding on every debounced change. It
// always calls stop exactly once before returning.
func (l *runLoop) run(ctx context.Context, backendRoot string, window time.Duration) error {
	l.onChange(ctx)

	watchStop, err := Watch(backendRoot, window, func() { l.onChange(ctx) })
	if err != nil {
		return fmt.Errorf("starting watcher: %w", err)
	}

	<-ctx.Done()
	watchStop()
	return l.stop()
}

// Run resolves the project root by walking up from cwd to the
// nearest ancestor go.mod, then runs the watch-build-restart loop
// against <root>/backend until ctx is cancelled.
func Run(ctx context.Context, fsys fs.FS, cwd string) error {
	root, err := migrate.ResolveRoot(cwd)
	if err != nil {
		return fmt.Errorf("resolving project root: %w", err)
	}
	modulePath, err := migrate.ModulePath(root)
	if err != nil {
		return fmt.Errorf("reading module path: %w", err)
	}

	backendRoot := filepath.Join(root, "backend")
	if info, err := os.Stat(backendRoot); err != nil || !info.IsDir() {
		return fmt.Errorf("no backend/ directory found at %s", root)
	}

	sup := NewSupervisor()
	binary := filepath.Join(root, BinaryPath)

	l := &runLoop{
		rebuild: func(ctx context.Context) error {
			if err := RegenerateMain(fsys, root, modulePath); err != nil {
				return err
			}
			return Build(ctx, root)
		},
		restart: func() error {
			return sup.Start(binary, os.Environ())
		},
		stop: sup.Stop,
	}

	return l.run(ctx, backendRoot, debounceWindow)
}
