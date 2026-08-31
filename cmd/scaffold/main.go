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
	case "migrate":
		os.Exit(runMigrate(os.Args[2:]))
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: gonext init <name> [path]")
	fmt.Fprintln(os.Stderr, "       gonext migrate")
}
