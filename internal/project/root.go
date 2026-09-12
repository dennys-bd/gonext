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

// ResolveRoot walks up from start looking for the nearest ancestor
// directory containing a go.mod file, mirroring how the go tool
// itself resolves the current module's root.
func ResolveRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, goModFilename)); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s found in %q or any parent directory", goModFilename, start)
		}
		dir = parent
	}
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
