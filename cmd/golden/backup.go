package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
)

// goldenDirName and goldenOldDirName are the canonical golden tree
// and its default backup, both resolved relative to the repo root
// (the directory `make golden` / `go run ./cmd/golden` is invoked
// from).
const (
	goldenDirName    = "golden"
	goldenOldDirName = "golden-old"
)

// backupAction is the decision planBackup makes about what to do
// with an existing golden/ tree before regenerating it.
type backupAction int

const (
	// actionGenerate means golden/ doesn't exist yet: generate
	// directly, nothing to back up.
	actionGenerate backupAction = iota
	// actionBackupThenGenerate means golden/ exists and golden-old/
	// doesn't: move golden/ to golden-old/, then generate.
	actionBackupThenGenerate
	// actionNeedsPrompt means both golden/ and golden-old/ already
	// exist: ask the developer whether to overwrite golden-old/ or
	// keep it and give the current golden/ a different backup name.
	actionNeedsPrompt
)

// planBackup decides what backupAction to take, given whether
// golden/ and golden-old/ currently exist. It is pure so the three
// cases described in the design doc are unit-testable without
// touching the filesystem.
func planBackup(goldenExists, goldenOldExists bool) backupAction {
	if !goldenExists {
		return actionGenerate
	}
	if !goldenOldExists {
		return actionBackupThenGenerate
	}
	return actionNeedsPrompt
}

// backupNamePattern mirrors scaffold.ValidateSlug's character rules:
// a backup name becomes a `golden-<name>` directory, so it must be
// safe to use as a single path segment.
var backupNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// errInvalidBackupName is the sentinel wrapped by validateBackupName.
var errInvalidBackupName = errors.New("invalid backup name")

// validateBackupName reports whether name is safe to use as the
// `golden-<name>` backup directory suffix: lowercase letters, digits,
// and hyphens only, and it must not collide with the reserved
// `golden`/`golden-old` directory names.
func validateBackupName(name string) error {
	if !backupNamePattern.MatchString(name) {
		return fmt.Errorf("%w: %q must contain only lowercase letters, digits, and hyphens, and start with a lowercase letter", errInvalidBackupName, name)
	}
	if name == goldenDirName || name == goldenOldDirName {
		return fmt.Errorf("%w: %q is a reserved name", errInvalidBackupName, name)
	}
	return nil
}

// dirExists reports whether path exists and is a directory.
func dirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// backupThenGenerate moves golden/ to golden-old/, making room for a
// fresh generation.
func backupThenGenerate() error {
	if err := os.Rename(goldenDirName, goldenOldDirName); err != nil {
		return fmt.Errorf("backing up %s to %s: %w", goldenDirName, goldenOldDirName, err)
	}
	return nil
}

// overwriteOldBackup discards the existing golden-old/ and replaces
// it with the current golden/.
func overwriteOldBackup() error {
	if err := os.RemoveAll(goldenOldDirName); err != nil {
		return fmt.Errorf("removing existing %s: %w", goldenOldDirName, err)
	}
	return backupThenGenerate()
}

// keepOldBackup leaves golden-old/ untouched and instead renames the
// current golden/ to golden-<name>.
func keepOldBackup(name string) error {
	if err := validateBackupName(name); err != nil {
		return err
	}
	target := fmt.Sprintf("%s-%s", goldenDirName, name)
	if exists, err := dirExists(target); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("%s already exists, choose a different name", target)
	}
	if err := os.Rename(goldenDirName, target); err != nil {
		return fmt.Errorf("renaming %s to %s: %w", goldenDirName, target, err)
	}
	return nil
}

// resolveBackupPrompt asks the developer, via promptFn, whether to
// overwrite the existing golden-old/ or keep it and give the current
// golden/ a different backup name (collected via promptNameFn), then
// applies the chosen action.
func resolveBackupPrompt(promptFn func() (overwrite bool, err error), promptNameFn func() (string, error)) error {
	overwrite, err := promptFn()
	if err != nil {
		return err
	}
	if overwrite {
		return overwriteOldBackup()
	}
	name, err := promptNameFn()
	if err != nil {
		return err
	}
	return keepOldBackup(name)
}
