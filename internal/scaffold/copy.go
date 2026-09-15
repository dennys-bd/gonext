package scaffold

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

const projectNameToken = "[PROJECT-NAME]"

// ModulePath is gonext itself, both the CLI and the core library a
// generated project imports at runtime.
const ModulePath = "github.com/dennys-bd/gonext"

// ModuleVersion is the gonext version `gonext init` pins into generated
// projects; bump it in the same change that tags a new vX.Y.Z. It is a
// pseudo-version until the first release is tagged.
const ModuleVersion = "v0.0.0-20260912225425-d0ae2746462c"

// binarySniffLen is how many leading bytes are inspected to decide
// whether a file is binary, matching the heuristic Git itself uses.
const binarySniffLen = 8192

// generatedFileMode is the mode every file Copy writes gets, since
// embed.FS always reports its entries as read-only.
const generatedFileMode = 0o644

// Copy walks the tree rooted at root within fsys and writes every file to
// dest, substituting [PROJECT-NAME] with slug in text files. agents selects
// which agent-tool-owned paths (see agentPaths) get written.
func Copy(fsys fs.FS, root, dest, slug string, agents []string) error {
	selected := make(map[string]bool, len(agents))
	for _, agent := range agents {
		selected[agent] = true
	}

	return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)

		if skipsAgentPath(filepath.ToSlash(rel), selected) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		// go.mod/go.sum are always regenerated fresh in the
		// destination via `go mod init`/`go mod tidy`, never copied
		// from the template source.
		if name := d.Name(); name == "go.mod" || name == "go.sum" {
			return nil
		}

		return copyFile(fsys, path, target, slug)
	})
}

// copyFile writes the file at path within fsys to target, substituting
// the [PROJECT-NAME] token in text files and copying binaries verbatim.
func copyFile(fsys fs.FS, path, target, slug string) error {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return err
	}

	if !isBinary(data) {
		data = bytes.ReplaceAll(data, []byte(projectNameToken), []byte(slug))
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, generatedFileMode)
}

// agentsDoc marks a directory as a gonext project for AddAgents:
// every generated project gets it unconditionally.
const agentsDoc = "AGENTS.md"

// AddAgents writes only the paths agents own (see agentPaths) into the
// existing project at dest, which must contain AGENTS.md. Nothing is written
// if any destination file already exists (without force) or is a symlink.
func AddAgents(fsys fs.FS, root, dest, slug string, agents []string, force bool) ([]string, error) {
	if _, err := os.Stat(filepath.Join(dest, agentsDoc)); err != nil {
		return nil, fmt.Errorf("no %s in %s: not a gonext project", agentsDoc, dest)
	}

	var files []string
	for _, agent := range agents {
		for _, owned := range agentPaths[agent] {
			err := fs.WalkDir(fsys, path.Join(root, owned), func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				rel, err := filepath.Rel(root, p)
				if err != nil {
					return err
				}
				files = append(files, filepath.ToSlash(rel))
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("adding %s: %w", owned, err)
			}
		}
	}

	var existing, symlinks []string
	for _, rel := range files {
		if link, ok := symlinkComponent(dest, rel); ok {
			if !slices.Contains(symlinks, link) {
				symlinks = append(symlinks, link)
			}
			continue
		}
		if _, err := os.Lstat(filepath.Join(dest, filepath.FromSlash(rel))); err == nil {
			existing = append(existing, rel)
		}
	}
	if len(symlinks) > 0 {
		return nil, fmt.Errorf("refusing to write through symlinks: %s", strings.Join(symlinks, ", "))
	}
	if len(existing) > 0 && !force {
		return nil, fmt.Errorf("refusing to overwrite existing files (pass --force to replace them): %s", strings.Join(existing, ", "))
	}

	for _, rel := range files {
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if err := copyFile(fsys, path.Join(root, rel), target, slug); err != nil {
			return nil, err
		}
	}
	return files, nil
}

// symlinkComponent reports the first component of rel under dest that is a
// symlink, if any — followed by MkdirAll/WriteFile it could land the write
// outside the project.
func symlinkComponent(dest, rel string) (string, bool) {
	parts := strings.Split(rel, "/")
	for i := range parts {
		sub := strings.Join(parts[:i+1], "/")
		fi, err := os.Lstat(filepath.Join(dest, filepath.FromSlash(sub)))
		if err != nil {
			// Missing from here down: nothing left to follow.
			return "", false
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return sub, true
		}
	}
	return "", false
}

// isBinary reports whether data looks like a binary file, using the
// same NUL-byte-in-the-prefix heuristic Git uses.
func isBinary(data []byte) bool {
	n := len(data)
	if n > binarySniffLen {
		n = binarySniffLen
	}
	return bytes.IndexByte(data[:n], 0) != -1
}
