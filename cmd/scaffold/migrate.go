package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dennys-bd/gonext/internal/migrate"
	"github.com/dennys-bd/gonext/internal/project"
)

const migrateUsage = "usage: gonext migrate [<domain>/<version>] [--yes]"

// runMigrate implements `gonext migrate`. With no target it applies every
// pending migration; with `<domain>/<version>` it brings that domain to
// that version, confirming a non-empty rollback unless --yes.
func runMigrate(args []string) int {
	target, yes, err := parseMigrateArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	root, err := project.Root(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	if err := project.LoadEnv(root); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	ctx := context.Background()
	if target == "" {
		err = migrate.Apply(ctx, root)
	} else {
		err = migrate.Migrate(ctx, root, target, yes)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

// parseMigrateArgs splits args into an optional positional target and the
// --yes flag, which may appear anywhere among them.
func parseMigrateArgs(args []string) (target string, yes bool, err error) {
	for _, arg := range args {
		if arg == "--yes" {
			yes = true
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return "", false, fmt.Errorf("unknown flag %q", arg)
		}
		if target != "" {
			return "", false, errors.New(migrateUsage)
		}
		target = arg
	}
	return target, yes, nil
}
