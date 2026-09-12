package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dennys-bd/gonext/internal/migrate"
	"github.com/dennys-bd/gonext/internal/project"
)

// runMigrate implements `gonext migrate` and returns the process
// exit code. It applies the pending Postgres migrations for the
// generated project in the current directory.
func runMigrate(args []string) int {
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

	if err := migrate.Apply(context.Background(), root); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}
