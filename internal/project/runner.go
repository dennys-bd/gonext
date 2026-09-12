package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// ModulePathToken is the placeholder runner templates use where the
// target project's module path goes.
const ModulePathToken = "[MODULE-PATH]"

const runnerFileMode = 0o644

// MaterializeRunner writes template — with ModulePathToken replaced
// by root's module path — to root/backend/filename and returns the
// backend directory together with a remove func the caller defers.
//
// The runner goes under backend/ rather than root so it can import
// backend's internal/ packages; Go's internal-import rule would block
// a file at root. filename must not start with "." or "_": the go
// tool silently excludes such files, which makes `go run` report "no
// Go files" instead of running it.
func MaterializeRunner(root, filename string, template []byte) (backendDir string, remove func(), err error) {
	modulePath, err := ModulePath(root)
	if err != nil {
		return "", nil, fmt.Errorf("resolving module path: %w", err)
	}

	backendDir = filepath.Join(root, "backend")
	runnerPath := filepath.Join(backendDir, filename)
	content := bytes.ReplaceAll(template, []byte(ModulePathToken), []byte(modulePath))
	if err := os.WriteFile(runnerPath, content, runnerFileMode); err != nil {
		return "", nil, fmt.Errorf("writing %s: %w", filename, err)
	}
	return backendDir, func() { _ = os.Remove(runnerPath) }, nil
}
