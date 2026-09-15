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

const generatePageUsage = "usage: gonext generate page <route> <operationId>"

const generateUsage = "usage: gonext generate [--check]\n       gonext generate migration <domain> <name>\n       gonext generate page <route> <operationId>"

// generateMode is parseGenerateArgs' verdict on `gonext generate`'s
// arguments.
type generateMode int

const (
	generateRun generateMode = iota
	generateCheck
	generateMigration
	generatePage
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
	if mode == generatePage {
		return runGeneratePage(args[1:])
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

// parseGenerateArgs classifies `gonext generate`'s arguments: no arguments,
// `--check`, a leading `migration` or `page`, or anything else (a usage
// error).
func parseGenerateArgs(args []string) (generateMode, error) {
	if len(args) > 0 && args[0] == "migration" {
		return generateMigration, nil
	}
	if len(args) > 0 && args[0] == "page" {
		return generatePage, nil
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
	domain, name, err := parseTwoPositionals(args, generateMigrationUsage)
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

// runGeneratePage implements `gonext generate page <route>
// <operationId>` for the generated project in the current directory.
func runGeneratePage(args []string) int {
	route, opID, err := parseTwoPositionals(args, generatePageUsage)
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

	paths, err := generate.Page(root, route, opID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	for _, p := range paths {
		fmt.Println("created", p)
	}
	return 0
}

// parseTwoPositionals requires exactly two positional arguments; any
// `--`-prefixed argument is unknown.
func parseTwoPositionals(args []string, usage string) (a, b string, err error) {
	var positional []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			return "", "", fmt.Errorf("unknown flag %q", arg)
		}
		positional = append(positional, arg)
	}
	if len(positional) != 2 {
		return "", "", errors.New(usage)
	}
	return positional[0], positional[1], nil
}
