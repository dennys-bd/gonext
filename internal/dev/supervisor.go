package dev

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// DefaultStopTimeout is how long Supervisor waits after SIGTERM
// before falling back to SIGKILL. It's set comfortably longer than
// templates/backend's default ShutdownTimeout (10s, see
// templates/backend/internal/config/config.go) so a normal graceful
// shutdown always has room to finish first.
const DefaultStopTimeout = 15 * time.Second

// Option configures a Supervisor.
type Option func(*Supervisor)

// WithStopTimeout overrides DefaultStopTimeout.
func WithStopTimeout(d time.Duration) Option {
	return func(s *Supervisor) { s.stopTimeout = d }
}

// Supervisor owns a single running child process, restarting it on
// demand with a graceful stop of the previous instance first.
type Supervisor struct {
	stopTimeout time.Duration

	mu  sync.Mutex
	cmd *exec.Cmd
}

// NewSupervisor returns a Supervisor with DefaultStopTimeout, or as
// overridden by opts.
func NewSupervisor(opts ...Option) *Supervisor {
	s := &Supervisor{stopTimeout: DefaultStopTimeout}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start stops any currently running child (see Stop), then starts
// binary as a new child process with inherited stdio and env.
func (s *Supervisor) Start(binary string, env []string, args ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.stopLocked(); err != nil {
		return err
	}

	cmd := exec.Command(binary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", binary, err)
	}
	s.cmd = cmd
	return nil
}

// Stop gracefully terminates the currently running child, if any:
// SIGTERM, then wait up to stopTimeout before falling back to
// SIGKILL.
func (s *Supervisor) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopLocked()
}

func (s *Supervisor) stopLocked() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	proc := s.cmd.Process
	cmd := s.cmd
	s.cmd = nil

	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()

	if err := proc.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("sending SIGTERM: %w", err)
	}

	select {
	case <-done:
	case <-time.After(s.stopTimeout):
		if err := proc.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("sending SIGKILL: %w", err)
		}
		<-done
	}
	return nil
}
