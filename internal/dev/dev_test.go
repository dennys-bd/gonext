package dev

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunLoop_OnChange_SuccessfulRebuildRestarts(t *testing.T) {
	var rebuildCalls, restartCalls int
	l := &runLoop{
		rebuild: func(ctx context.Context) error { rebuildCalls++; return nil },
		restart: func() error { restartCalls++; return nil },
	}

	l.onChange(context.Background())

	if rebuildCalls != 1 {
		t.Errorf("rebuild calls = %d, want 1", rebuildCalls)
	}
	if restartCalls != 1 {
		t.Errorf("restart calls = %d, want 1", restartCalls)
	}
}

func TestRunLoop_OnChange_BuildFailureLeavesPriorProcessRunning(t *testing.T) {
	var restartCalls int
	l := &runLoop{
		rebuild: func(ctx context.Context) error { return errors.New("compile error") },
		restart: func() error { restartCalls++; return nil },
	}

	l.onChange(context.Background())

	if restartCalls != 0 {
		t.Errorf("restart calls = %d, want 0 (a failed build must never touch the running process)", restartCalls)
	}
}

func TestRunLoop_Run_ExitsOnCancellation(t *testing.T) {
	backendRoot := t.TempDir()

	var rebuildCalls, restartCalls, stopCalls int
	l := &runLoop{
		rebuild: func(ctx context.Context) error { rebuildCalls++; return nil },
		restart: func() error { restartCalls++; return nil },
		stop:    func() error { stopCalls++; return nil },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := l.run(ctx, backendRoot, 20*time.Millisecond)
	if err != nil {
		t.Errorf("run: unexpected error: %v", err)
	}
	if rebuildCalls == 0 {
		t.Errorf("expected at least one initial rebuild before cancellation")
	}
	if restartCalls == 0 {
		t.Errorf("expected at least one initial restart before cancellation")
	}
	if stopCalls != 1 {
		t.Errorf("stop calls = %d, want 1 on cancellation", stopCalls)
	}
}

func TestRunLoop_Run_RebuildsOnFileChange(t *testing.T) {
	backendRoot := t.TempDir()

	rebuilds := make(chan struct{}, 8)
	l := &runLoop{
		rebuild: func(ctx context.Context) error { rebuilds <- struct{}{}; return nil },
		restart: func() error { return nil },
		stop:    func() error { return nil },
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- l.run(ctx, backendRoot, 20*time.Millisecond) }()

	// Drain the initial rebuild.
	select {
	case <-rebuilds:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected an initial rebuild")
	}

	time.Sleep(100 * time.Millisecond) // let the watcher finish registering
	if err := os.WriteFile(filepath.Join(backendRoot, "app.go"), []byte("package backend\n"), 0o644); err != nil {
		t.Fatalf("writing app.go: %v", err)
	}

	select {
	case <-rebuilds:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected a rebuild triggered by the file change")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("run: unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("run did not exit after cancellation")
	}
}
