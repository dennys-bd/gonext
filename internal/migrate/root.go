// Package migrate applies a generated project's pending Postgres
// migrations from this repo, without vendoring the migrate command
// into every scaffolded project (see templates/backend/cmd/migrate,
// which this package replaces).
package migrate

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
