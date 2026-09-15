package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dennys-bd/gonext/internal/generate"
	"github.com/dennys-bd/gonext/internal/project"
)

const generateMigrationUsage = "usage: gonext generate migration <domain> <name>"

const generateUsage = "usage: gonext generate [--check]\n       gonext generate migration <domain> <name>"

// generateMode is parseGenerateArgs' verdict on `gonext generate`'s
// arguments.
type generateMode int

const (
	generateRun generateMode = iota
	generateCheck
	generateMigration
)

// runGenerate implements `gonext generate [--check]` and
// `gonext generate migration <domain> <name>`, returning the process
// exit code.
func runGenerate(args []string) int {
	mode, err := parseGenerateArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if mode == generateMigration {
		return runGenerateMigration(args[1:])
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

	check := mode == generateCheck
	if err := generate.RunAll(context.Background(), root, check); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	if check {
		fmt.Println("up to date")
	} else {
		fmt.Println("regenerated")
	}
	return 0
}

// parseGenerateArgs classifies `gonext generate`'s arguments: no
// arguments runs every step, `--check` checks every step, a leading
// `migration` defers to the reserved migration subcommand, and
// anything else is a usage error.
func parseGenerateArgs(args []string) (generateMode, error) {
	if len(args) > 0 && args[0] == "migration" {
		return generateMigration, nil
	}
	if len(args) == 0 {
		return generateRun, nil
	}
	if len(args) == 1 && args[0] == "--check" {
		return generateCheck, nil
	}
	return generateRun, errors.New(generateUsage)
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
