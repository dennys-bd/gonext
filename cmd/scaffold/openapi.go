package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dennys-bd/gonext/internal/openapi"
	"github.com/dennys-bd/gonext/internal/project"
)

// runOpenAPI implements `gonext openapi [--check]` and returns the
// process exit code. It regenerates the generated project's
// docs/openapi.yaml and typed frontend client, or with --check only
// reports whether the committed document is stale.
func runOpenAPI(args []string) int {
	check := len(args) == 1 && args[0] == "--check"
	if len(args) > 0 && !check {
		fmt.Fprintln(os.Stderr, "usage: gonext openapi [--check]")
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

	run := openapi.Write
	if check {
		run = openapi.Check
	}
	if err := run(context.Background(), root); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}
