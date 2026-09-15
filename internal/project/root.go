// Package project locates a generated project's root and materializes the
// temporary runner files CLI-native subcommands `go run` inside it.
package project

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const goModFilename = "go.mod"

// Root checks that dir holds a go.mod and returns it as an absolute path. It
// does not search parent directories: subcommands must be run from the
// project root.
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
