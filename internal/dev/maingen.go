// Package dev implements `gonext dev`: a watch-build-restart loop for
// a generated project's backend/, run from this repo's own binary
// rather than vendored into templates/.
package dev

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	projectNameToken   = "[PROJECT-NAME]"
	mainGoTemplatePath = "templates/backend/main.go"
	generatedFileMode  = 0o644
)

// RegenerateMain overwrites <root>/backend/main.go with the canonical
// template read from fsys, substituting modulePath for the
// [PROJECT-NAME] token. It unconditionally overwrites any existing
// content — there is no diff/hand-edit detection by design.
func RegenerateMain(fsys fs.FS, root, modulePath string) error {
	data, err := fs.ReadFile(fsys, mainGoTemplatePath)
	if err != nil {
		return fmt.Errorf("reading embedded %s: %w", mainGoTemplatePath, err)
	}
	data = bytes.ReplaceAll(data, []byte(projectNameToken), []byte(modulePath))

	dest := filepath.Join(root, "backend", "main.go")

	// Skip the write when content is already up to date. gonext dev
	// calls RegenerateMain before every build, including rebuilds the
	// watcher itself triggered; backend/main.go is a watched .go file,
	// so an unconditional rewrite here would touch its mtime on every
	// single build and re-trigger another rebuild — an infinite loop.
	if existing, err := os.ReadFile(dest); err == nil && bytes.Equal(existing, data) {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(dest), err)
	}
	if err := os.WriteFile(dest, data, generatedFileMode); err != nil {
		return fmt.Errorf("writing %s: %w", dest, err)
	}
	return nil
}
