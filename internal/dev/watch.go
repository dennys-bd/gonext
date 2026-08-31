package dev

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// excludedDir is the one backend subdirectory whose contents never
// trigger a rebuild — it holds gonext dev's own compiled output.
const excludedDir = "bin"

// shouldTrigger reports whether a change at path (a descendant of
// backendRoot) should cause a rebuild: a .go file not under
// backend/bin/.
func shouldTrigger(backendRoot, path string) bool {
	if filepath.Ext(path) != ".go" {
		return false
	}
	rel, err := filepath.Rel(backendRoot, path)
	if err != nil {
		return false
	}
	first, _, _ := strings.Cut(rel, string(filepath.Separator))
	return first != excludedDir
}

// Debouncer collapses a burst of Trigger calls arriving within window
// of each other into a single value delivered on C.
type Debouncer struct {
	C      chan struct{}
	window time.Duration

	mu    sync.Mutex
	timer *time.Timer
}

// NewDebouncer returns a Debouncer that waits window after the last
// Trigger call before delivering a signal on C.
func NewDebouncer(window time.Duration) *Debouncer {
	return &Debouncer{C: make(chan struct{}, 1), window: window}
}

// Trigger (re)starts the debounce window; a signal is delivered on C
// only once no further Trigger call arrives for window.
func (d *Debouncer) Trigger() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.window, func() {
		select {
		case d.C <- struct{}{}:
		default:
		}
	})
}

// Stop cancels any pending debounced signal.
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
	}
}

// Watch recursively watches backendRoot for .go file changes
// (excluding backend/bin/), debouncing bursts within window into a
// single call to onChange. It returns a stop function that shuts the
// watcher down; onChange is never called after stop returns.
func Watch(backendRoot string, window time.Duration, onChange func()) (stop func(), err error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating watcher: %w", err)
	}

	if err := registerTree(w, backendRoot); err != nil {
		w.Close()
		return nil, fmt.Errorf("registering %s: %w", backendRoot, err)
	}

	debouncer := NewDebouncer(window)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case event, ok := <-w.Events:
				if !ok {
					return
				}
				handleEvent(w, backendRoot, event, debouncer)
			case <-w.Errors:
				// Watch errors are non-fatal to the loop; the
				// watcher keeps running on remaining directories.
			}
		}
	}()

	go func() {
		for {
			select {
			case <-done:
				return
			case _, ok := <-debouncer.C:
				if !ok {
					return
				}
				onChange()
			}
		}
	}()

	stop = func() {
		close(done)
		debouncer.Stop()
		w.Close()
	}
	return stop, nil
}

// handleEvent registers newly created directories and forwards
// qualifying file events to the debouncer.
func handleEvent(w *fsnotify.Watcher, backendRoot string, event fsnotify.Event, debouncer *Debouncer) {
	if event.Op&fsnotify.Create != 0 {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			_ = w.Add(event.Name)
		}
	}

	if shouldTrigger(backendRoot, event.Name) {
		debouncer.Trigger()
	}
}

// registerTree walks root and registers every directory with w,
// skipping backend/bin/.
func registerTree(w *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == excludedDir && path != root {
			return filepath.SkipDir
		}
		return w.Add(path)
	})
}
