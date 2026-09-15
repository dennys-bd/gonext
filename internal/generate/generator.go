package generate

import (
	"context"
	"fmt"

	xexec "github.com/dennys-bd/gonext/internal/exec"
)

// Generator is one codegen step `gonext generate` runs from the project
// root. Check may be nil, in which case `--check` skips the step.
type Generator struct {
	Name       string
	Run, Check []string

	// stale, when set, is a more specific error than "<name>: <err>" for
	// this step's Check failure.
	stale string
}

// Generators returns the project's codegen steps in execution order. root is
// accepted unused for a future config file to read the step list from.
func Generators(root string) []Generator {
	return []Generator{
		{
			Name:  "wire",
			Run:   []string{"go", "tool", "wire", "./backend/..."},
			Check: []string{"go", "tool", "wire", "diff", "./backend/..."},
			stale: "wire_gen.go is stale (run gonext generate)",
		},
	}
}

// runFunc executes one step's argv; overridden in tests to avoid
// invoking `go tool wire` for real.
var runFunc = xexec.Run

// RunAll runs (or, when check is true, checks) every step Generators
// returns for root, in root's working directory, stopping at the
// first failure.
func RunAll(ctx context.Context, root string, check bool) error {
	return runGenerators(ctx, root, Generators(root), check)
}

// runGenerators is RunAll's implementation over an explicit step
// list, so tests can exercise the nil-Check skip and stop-on-first-
// failure behaviour without widening Generators itself.
func runGenerators(ctx context.Context, root string, gens []Generator, check bool) error {
	for _, g := range gens {
		argv := g.Run
		if check {
			argv = g.Check
			if argv == nil {
				continue
			}
		}
		if err := runFunc(ctx, root, argv[0], argv[1:]...); err != nil {
			if check && g.stale != "" {
				return fmt.Errorf("%s: %w", g.stale, err)
			}
			return fmt.Errorf("%s: %w", g.Name, err)
		}
	}
	return nil
}
