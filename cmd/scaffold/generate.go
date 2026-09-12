package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dennys-bd/gonext/internal/generate"
	"github.com/dennys-bd/gonext/internal/project"
)

const generateMigrationUsage = "usage: gonext generate migration <domain> <name>"

// runGenerate implements `gonext generate <target> ...` and returns
// the process exit code. Today the only target is migration.
func runGenerate(args []string) int {
	if len(args) == 0 || args[0] != "migration" {
		fmt.Fprintln(os.Stderr, generateMigrationUsage)
		return 1
	}
	return runGenerateMigration(args[1:])
}

// runGenerateMigration implements `gonext generate migration
// <domain> <name>` for the generated project in the current directory.
func runGenerateMigration(args []string) int {
	domain, name, err := parseGenerateMigrationArgs(args)
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

	rel, err := generate.Migration(root, domain, name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	fmt.Println("created", rel)
	return 0
}

// parseGenerateMigrationArgs requires exactly two positionals; any
// `--`-prefixed argument is unknown.
func parseGenerateMigrationArgs(args []string) (domain, name string, err error) {
	var positional []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			return "", "", fmt.Errorf("unknown flag %q", arg)
		}
		positional = append(positional, arg)
	}
	if len(positional) != 2 {
		return "", "", errors.New(generateMigrationUsage)
	}
	return positional[0], positional[1], nil
}
