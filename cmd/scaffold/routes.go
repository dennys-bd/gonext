package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/dennys-bd/gonext/internal/generate"
	"github.com/dennys-bd/gonext/internal/project"
)

const routesUsage = "usage: gonext routes"

// runRoutes implements `gonext routes` for the generated project in
// the current directory, returning the process exit code.
func runRoutes(args []string) int {
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, routesUsage)
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

	routes, err := generate.Routes(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	printRoutes(os.Stdout, routes)
	return 0
}

// printRoutes writes routes to w as a tab-aligned table, one row per
// operation.
func printRoutes(w io.Writer, routes []generate.Route) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "operationId\tmethod\tpath\tclient call\tused by")
	for _, r := range routes {
		usedBy := "—"
		if len(r.UsedBy) > 0 {
			usedBy = strings.Join(r.UsedBy, ", ")
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", r.OperationID, r.Method, r.Path, r.CallName, usedBy)
	}
	tw.Flush()
}
