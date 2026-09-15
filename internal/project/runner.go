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

// MaterializeRunner writes template (extra's tokens replaced before
// ModulePathToken) to root/backend/filename, returning the backend dir and a
// remove func to defer. filename must not start with "." or "_" (go run ignores such files).
func MaterializeRunner(root, filename string, template []byte, extra map[string]string) (backendDir string, remove func(), err error) {
	modulePath, err := ModulePath(root)
	if err != nil {
		return "", nil, fmt.Errorf("resolving module path: %w", err)
	}

	backendDir = filepath.Join(root, "backend")
	runnerPath := filepath.Join(backendDir, filename)
	content := template
	for token, value := range extra {
		content = bytes.ReplaceAll(content, []byte(token), []byte(value))
	}
	content = bytes.ReplaceAll(content, []byte(ModulePathToken), []byte(modulePath))
	if err := os.WriteFile(runnerPath, content, runnerFileMode); err != nil {
		return "", nil, fmt.Errorf("writing %s: %w", filename, err)
	}
	return backendDir, func() { _ = os.Remove(runnerPath) }, nil
}
