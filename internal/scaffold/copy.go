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

// ModulePath is gonext itself. It is both the CLI and the core
// library a generated project imports at runtime (today
// github.com/dennys-bd/gonext/auth), so a project depends on the one
// module rather than on a separately versioned contract.
const ModulePath = "github.com/dennys-bd/gonext"

// ModuleVersion is the gonext version `gonext init` pins into
// generated projects. Bump it in the same change that tags a new
// vX.Y.Z, so a project keeps building against the library its
// scaffold was written for rather than drifting onto `latest`.
//
// It is a pseudo-version until the first release is tagged: gonext is
// resolvable from any pushed commit, so generated projects build
// today without waiting on a tag.
const ModuleVersion = "v0.0.0-20260901030717-52ea7aa89005"

// binarySniffLen is how many leading bytes are inspected to decide
// whether a file is binary, matching the heuristic Git itself uses.
const binarySniffLen = 8192

// generatedFileMode is the permission mode every file Copy writes
// gets, regardless of the source's mode. embed.FS always reports its
// entries as read-only (0444), and propagating that mode verbatim
// would make every file in a generated project unwritable.
const generatedFileMode = 0o644

// Copy walks the tree rooted at root within fsys and writes every
// file to dest. Text files have every occurrence of the
// [PROJECT-NAME] token replaced with slug; files detected as binary
// are copied verbatim. agents selects which agent-tool-owned paths
// (see agentPaths) are written; a path owned by no tool is always
// written regardless of agents.
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

// AddAgents writes only the paths agents own (see agentPaths) from
// the template tree rooted at root within fsys into the existing
// project at dest, substituting slug exactly as Copy does. dest must
// contain AGENTS.md. Every destination file is checked before any is
// written: if one already exists and force is false, the error
// names all of them and nothing is written; a symlink at any of them
// is refused even with force. It returns the
// dest-relative, slash-separated paths written, in walk order.
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

// symlinkComponent reports the first component of rel under dest that
// is a symlink, if any. A symlink anywhere on the path — a leaf like
// CLAUDE.md or a directory like .claude, planted by a booby-trapped
// checkout or pointing at a dotfile — would otherwise be followed by
// MkdirAll/WriteFile and land the file outside the project.
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
