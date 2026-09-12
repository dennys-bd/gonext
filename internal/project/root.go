// Package project locates a generated project from the current
// directory and materializes the temporary runner files that
// CLI-native subcommands (gonext migrate, gonext openapi) `go run`
// inside it — so those capabilities live in this repo instead of
// being vendored into every scaffolded project.
package project

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const goModFilename = "go.mod"

// Root checks that dir is a generated project's root — the directory
// holding its go.mod, where `gonext init` ran `go mod init` — and
// returns it as an absolute path. Subcommands are meant to be run
// from the root (the generated Makefile does), so a go.mod is
// required right there rather than searched for in parent
// directories.
func Root(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path: %w", err)
	}
	if _, err := os.Stat(filepath.Join(abs, goModFilename)); err != nil {
		return "", fmt.Errorf("no %s in %s: run this from the project root", goModFilename, abs)
	}
	return abs, nil
}

// ModulePath reads root's go.mod and returns its module path.
func ModulePath(root string) (string, error) {
	f, err := os.Open(filepath.Join(root, goModFilename))
	if err != nil {
		return "", fmt.Errorf("opening %s: %w", goModFilename, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if after, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(after), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading %s: %w", goModFilename, err)
	}
	return "", fmt.Errorf("no module declaration found in %s", goModFilename)
}
