package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dennys-bd/gonext/internal/generate"
	"github.com/dennys-bd/gonext/internal/openapi"
	"github.com/dennys-bd/gonext/internal/project"
	"github.com/dennys-bd/gonext/internal/scaffold"
)

const generateMigrationUsage = "usage: gonext generate migration <domain> <name>"

const generatePageUsage = "usage: gonext generate page <route> <operationId>  (see gonext routes)"

const generateResourceUsage = "usage: gonext generate resource <domain> <name> [<field>:<type>…] [--ops create,get,list,update,delete] [--auth all|mutations|public] [--plural <word>]"

const generateUsage = "usage: gonext generate [--check]\n       gonext generate migration <domain> <name>\n       gonext generate page <route> <operationId>\n       gonext generate resource <domain> <name> [<field>:<type>…] [--ops <list>] [--auth all|mutations|public] [--plural <word>]"

// generateMode is parseGenerateArgs' verdict on `gonext generate`'s
// arguments.
type generateMode int

const (
	generateRun generateMode = iota
	generateCheck
	generateMigration
	generatePage
	generateResource
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
	if mode == generateResource {
		return runGenerateResource(args[1:])
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
	if len(args) > 0 && args[0] == "resource" {
		return generateResource, nil
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

// resourceArgs is parseGenerateResourceArgs' verdict: the spec built
// from the positionals and --auth/--plural, plus the raw --ops flag
// (opsSet distinguishes "not given" from "given empty").
type resourceArgs struct {
	spec    generate.ResourceSpec
	opsFlag string
	opsSet  bool
}

// parseGenerateResourceArgs parses `gonext generate resource`'s
// arguments: <domain> <name> [<field>:<type>…] in any order relative
// to --ops/--auth/--plural, each accepted as `--flag value` or
// `--flag=value`.
func parseGenerateResourceArgs(args []string) (resourceArgs, error) {
	var ra resourceArgs
	var positional []string
	for i := 0; i < len(args); i++ {
		name, value, hasValue := strings.Cut(args[i], "=")
		if !strings.HasPrefix(name, "--") {
			positional = append(positional, args[i])
			continue
		}
		if name != "--ops" && name != "--auth" && name != "--plural" {
			return resourceArgs{}, fmt.Errorf("unknown flag %q", name)
		}
		if !hasValue {
			i++
			if i >= len(args) {
				return resourceArgs{}, fmt.Errorf("flag %s needs a value", name)
			}
			value = args[i]
		}
		switch name {
		case "--ops":
			ra.opsFlag, ra.opsSet = value, true
		case "--auth":
			ra.spec.Auth = value
		case "--plural":
			ra.spec.Plural = value
		}
	}
	if len(positional) < 2 {
		return resourceArgs{}, errors.New(generateResourceUsage)
	}
	ra.spec.Domain = positional[0]
	ra.spec.Name = positional[1]
	if len(positional) > 2 {
		ra.spec.Fields = positional[2:]
	}
	return ra, nil
}

// resolveOps returns the operations gonext generate resource asks
// for: the parsed --ops flag verbatim when given (generate.Resource
// validates its contents), the interactive prompt's answer on a TTY,
// or nil (every operation) otherwise.
func resolveOps(opsFlag string, opsSet, isTTY bool, prompt func([]string) ([]string, error)) ([]string, error) {
	if opsSet {
		return strings.Split(opsFlag, ","), nil
	}
	if isTTY {
		return prompt(generate.AllOps)
	}
	return nil, nil
}

// runGenerateResource implements `gonext generate resource <domain>
// <name> [<field>:<type>…]` for the generated project in the current
// directory.
func runGenerateResource(args []string) int {
	ra, err := parseGenerateResourceArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	ra.spec.Ops, err = resolveOps(ra.opsFlag, ra.opsSet, scaffold.IsTTY(), scaffold.PromptOps)
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

	result, err := generate.Resource(root, ra.spec)
	for _, p := range result.Created {
		fmt.Println("created", p)
	}
	if result.Facade != "" {
		fmt.Println("updated", result.Facade)
	}
	for _, p := range result.Renumbered {
		fmt.Println("updated", p)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	// The refresh is an error, not a warning: the slice is on disk, but
	// a stale contract is a broken project.
	if err := openapi.Write(context.Background(), root); err != nil {
		fmt.Fprintln(os.Stderr, "error: refreshing the contract:", err)
		return 1
	}

	fmt.Printf("next: gonext migrate %s   # applies %s\n", ra.spec.Domain, result.Migration)
	return 0
}
