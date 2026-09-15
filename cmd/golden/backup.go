package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
)

// goldenDirName and goldenOldDirName are resolved relative to the
// repo root, the directory `make golden` is invoked from.
const (
	goldenDirName    = "golden"
	goldenOldDirName = "golden-old"
)

type backupAction int

const (
	actionGenerate backupAction = iota
	actionBackupThenGenerate
	actionNeedsPrompt
)

// planBackup is pure so it's unit-testable without touching the filesystem.
func planBackup(goldenExists, goldenOldExists bool) backupAction {
	if !goldenExists {
		return actionGenerate
	}
	if !goldenOldExists {
		return actionBackupThenGenerate
	}
	return actionNeedsPrompt
}

// backupNamePattern mirrors scaffold.ValidateSlug: the name becomes a
// `golden-<name>` path segment, so it must be safe as one.
var backupNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// errInvalidBackupName is the sentinel wrapped by validateBackupName.
var errInvalidBackupName = errors.New("invalid backup name")

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

// resolveBackupPrompt applies the developer's overwrite-or-rename choice
// from promptFn and promptNameFn to the existing golden-old/ backup.
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
