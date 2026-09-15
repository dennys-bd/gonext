// Command golden regenerates the committed golden/ dev tree by dogfooding
// `gonext init` against this repo's own templates/, backing up any existing
// golden/ first.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"

	xexec "github.com/dennys-bd/gonext/internal/exec"
)

// goldenSlug must stay in sync with the slug baked into golden's
// substituted content (see cmd/scaffold/copy_golden_test.go's
// goldenSlug).
const goldenSlug = "golden-app"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	goldenExists, err := dirExists(goldenDirName)
	if err != nil {
		return fmt.Errorf("checking %s: %w", goldenDirName, err)
	}
	goldenOldExists, err := dirExists(goldenOldDirName)
	if err != nil {
		return fmt.Errorf("checking %s: %w", goldenOldDirName, err)
	}

	switch planBackup(goldenExists, goldenOldExists) {
	case actionBackupThenGenerate:
		if err := backupThenGenerate(); err != nil {
			return err
		}
	case actionNeedsPrompt:
		if err := resolveBackupPrompt(promptOverwrite, promptBackupName); err != nil {
			return err
		}
	}

	ctx := context.Background()
	if err := xexec.Run(ctx, ".", "go", "run", "./cmd/scaffold", "init", goldenSlug, "./"+goldenDirName, "--agents=none"); err != nil {
		return fmt.Errorf("generating %s: %w", goldenDirName, err)
	}
	return nil
}

// promptOverwrite asks whether to overwrite golden-old/ or keep it under a
// different name.
func promptOverwrite() (bool, error) {
	var overwrite bool
	field := huh.NewConfirm().
		Title(fmt.Sprintf("%s already exists. Overwrite it with the current %s?", goldenOldDirName, goldenDirName)).
		Affirmative("Overwrite " + goldenOldDirName).
		Negative("Keep it, back up " + goldenDirName + " under a different name").
		Value(&overwrite)
	if err := huh.NewForm(huh.NewGroup(field)).Run(); err != nil {
		return false, err
	}
	return overwrite, nil
}

// promptBackupName asks for the golden-<name> suffix, re-prompting until
// validateBackupName accepts it.
func promptBackupName() (string, error) {
	var name string
	field := huh.NewInput().
		Title(fmt.Sprintf("Backup name for the current %s (used as %s-<name>)?", goldenDirName, goldenDirName)).
		Value(&name).
		Validate(validateBackupName)
	if err := huh.NewForm(huh.NewGroup(field)).Run(); err != nil {
		return "", err
	}
	return name, nil
}
