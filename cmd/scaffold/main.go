// Command gonext scaffolds new projects from the gonext templates.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		os.Exit(runInit(os.Args[2:]))
	case "add":
		os.Exit(runAdd(os.Args[2:]))
	case "generate":
		os.Exit(runGenerate(os.Args[2:]))
	case "migrate":
		os.Exit(runMigrate(os.Args[2:]))
	case "dev":
		os.Exit(runDev(os.Args[2:]))
	case "openapi":
		os.Exit(runOpenAPI(os.Args[2:]))
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: gonext init <name> [path] [--agents=<list>]")
	fmt.Fprintln(os.Stderr, "       gonext add agent <tool>... [--force]")
	fmt.Fprintln(os.Stderr, "       gonext generate migration <domain> <name>")
	fmt.Fprintln(os.Stderr, "       gonext migrate [<domain>/<version>] [--yes]")
	fmt.Fprintln(os.Stderr, "       gonext dev")
	fmt.Fprintln(os.Stderr, "       gonext openapi [--check]")
}
