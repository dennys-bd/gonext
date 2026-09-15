// Package generate implements the dev-loop generators behind
// `gonext generate`, writing files into an existing project.
package generate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
)

var migrationNameRE = regexp.MustCompile(`^[a-z0-9_]+$`)
var migrationFileRE = regexp.MustCompile(`^(\d{4})_`)

// Register's two arguments are anonymous, not named top-level up/down: every
// migration file in a domain shares one package, so named functions would
// collide across files.
const migrationSkeleton = `package migrations

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/dennys-bd/gonext/dbmigrate"
)

func init() {
	dbmigrate.Register(
		// Apply the schema change, e.g. db.ExecContext(ctx, "CREATE TABLE ...").
		func(ctx context.Context, db *bun.DB) error {
			return nil
		},
		// Undo what up did, e.g. db.ExecContext(ctx, "DROP TABLE ...").
		func(ctx context.Context, db *bun.DB) error {
			return nil
		},
	)
}
`

const fileMode = 0o644
const dirMode = 0o755

// Migration writes backend/<domain>/migrations/<NNNN>_<name>.go under
// root with the domain's next sequence number and returns its
// root-relative slash path. It never creates a domain.
func Migration(root, domain, name string) (string, error) {
	// The same character set dbmigrate's path regex accepts, which
	// also keeps a domain from naming a path outside backend/.
	if domain == "internal" || !migrationNameRE.MatchString(domain) {
		return "", fmt.Errorf("no such domain backend/%s", domain)
	}
	info, err := os.Stat(filepath.Join(root, "backend", domain))
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("no such domain backend/%s", domain)
	}
	if !migrationNameRE.MatchString(name) {
		return "", fmt.Errorf("invalid migration name %q: use snake_case", name)
	}

	dir := filepath.Join(root, "backend", domain, "migrations")
	version, err := nextVersion(dir)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, dirMode); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}
	filename := version + "_" + name + ".go"
	// O_EXCL: never truncate a file that appeared between the scan
	// and the write, or one the developer created by hand.
	f, err := os.OpenFile(filepath.Join(dir, filename), os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if err != nil {
		return "", fmt.Errorf("writing %s: %w", filename, err)
	}
	if _, err := f.WriteString(migrationSkeleton); err != nil {
		f.Close()
		return "", fmt.Errorf("writing %s: %w", filename, err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("writing %s: %w", filename, err)
	}

	return path.Join("backend", domain, "migrations", filename), nil
}

// nextVersion returns the four-digit sequence one past the highest
// NNNN among dir's *.go files, or "0001" when dir does not exist or
// holds no matching file.
func nextVersion(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return "0001", nil
	}
	if err != nil {
		// Any other failure must not restart the sequence at 0001 and
		// overwrite whatever is really there.
		return "", fmt.Errorf("reading %s: %w", dir, err)
	}

	highest := 0
	for _, entry := range entries {
		match := migrationFileRE.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}
		n, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		highest = max(highest, n)
	}
	return fmt.Sprintf("%04d", highest+1), nil
}
