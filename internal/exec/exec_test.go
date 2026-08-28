package exec

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestWaitHealthy_SucceedsImmediately(t *testing.T) {
	err := WaitHealthy(context.Background(), time.Second, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("WaitHealthy: unexpected error: %v", err)
	}
}

func TestWaitHealthy_SucceedsAfterRetries(t *testing.T) {
	attempts := 0
	err := WaitHealthy(context.Background(), 2*time.Second, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("not ready")
		}
		return nil
	})
	if err != nil {
		t.Errorf("WaitHealthy: unexpected error: %v", err)
	}
	if attempts < 3 {
		t.Errorf("expected at least 3 attempts, got %d", attempts)
	}
}

func TestWaitHealthy_TimesOut(t *testing.T) {
	start := time.Now()
	err := WaitHealthy(context.Background(), 200*time.Millisecond, func(ctx context.Context) error {
		return errors.New("never ready")
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("WaitHealthy: expected timeout error, got nil")
	}
	if elapsed > 2*time.Second {
		t.Errorf("WaitHealthy: took too long to time out: %v", elapsed)
	}
}
