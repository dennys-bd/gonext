package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	gonext "github.com/dennys-bd/gonext"
	"github.com/dennys-bd/gonext/internal/project"
	"github.com/dennys-bd/gonext/internal/scaffold"
)

const addAgentUsage = "usage: gonext add agent <tool>... [--force]"

// runAdd implements `gonext add <target> ...` and returns the process
// exit code. Today the only addable target is agent tooling.
func runAdd(args []string) int {
	if len(args) == 0 || args[0] != "agent" {
		fmt.Fprintln(os.Stderr, addAgentUsage)
		return 1
	}
	return runAddAgent(args[1:])
}

// runAddAgent implements `gonext add agent <tool>... [--force]`: it
// writes the named tools' files into the generated project in the
// current directory, refusing to overwrite existing ones unless
// --force is given.
func runAddAgent(args []string) int {
	names, force, err := parseAddAgentArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	agents, err := scaffold.ParseAgentNames(names)
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
	slug, err := project.ModulePath(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	written, err := scaffold.AddAgents(gonext.Templates, "templates", root, slug, agents, force)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	if len(written) == 0 {
		fmt.Println("nothing to add: AGENTS.md already covers", strings.Join(agents, ", "))
		return 0
	}
	for _, rel := range written {
		fmt.Println("added", rel)
	}
	return 0
}

// parseAddAgentArgs splits args into positional tool names and the
// --force flag, which may appear anywhere among them. Any other
// `--`-prefixed argument is an error; zero tool names is a usage error.
func parseAddAgentArgs(args []string) (names []string, force bool, err error) {
	for _, arg := range args {
		if arg == "--force" {
			force = true
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return nil, false, fmt.Errorf("unknown flag %q", arg)
		}
		names = append(names, arg)
	}
	if len(names) == 0 {
		return nil, false, errors.New(addAgentUsage)
	}
	return names, force, nil
}
