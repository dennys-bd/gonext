package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dennys-bd/gonext/internal/openapi"
	"github.com/dennys-bd/gonext/internal/project"
)

// runOpenAPI implements `gonext openapi [--check]`, regenerating
// docs/openapi.yaml and the frontend client, or just checking staleness.
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

	if check {
		if err := openapi.Check(context.Background(), root); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		fmt.Println(openapi.DocumentPath, "is up to date")
		return 0
	}
	if err := openapi.Write(context.Background(), root); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	fmt.Println("wrote", openapi.DocumentPath)
	return 0
}
