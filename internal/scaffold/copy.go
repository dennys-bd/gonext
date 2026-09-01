package scaffold

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
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
// are copied verbatim.
func Copy(fsys fs.FS, root, dest, slug string) error {
	return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		// go.mod/go.sum are always regenerated fresh in the
		// destination via `go mod init`/`go mod tidy`, never copied
		// from the template source.
		if name := d.Name(); name == "go.mod" || name == "go.sum" {
			return nil
		}

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
	})
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
