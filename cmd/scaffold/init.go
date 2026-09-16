package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gonext "github.com/dennys-bd/gonext"
	xexec "github.com/dennys-bd/gonext/internal/exec"
	"github.com/dennys-bd/gonext/internal/scaffold"
)

const (
	localConfig        = "mise.local.toml"
	localConfigExample = localConfig + ".example"
)

// runInit implements `gonext init [name] [path] [--agents=<list>]` and
// returns the process exit code.
func runInit(args []string) int {
	name, path, agentsArg, agentsSet, err := parseInitArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	var agents []string
	if agentsSet {
		agents, err = scaffold.ParseAgents(agentsArg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
	}

	slug, err := resolveSlug(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	if !agentsSet && scaffold.IsTTY() {
		agents, err = scaffold.PromptAgents()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
	}

	dest, err := scaffold.ResolveDest(slug, path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	if err := scaffold.CheckEmpty(dest); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	if err := scaffold.Copy(gonext.Templates, "templates", dest, slug, agents); err != nil {
		fmt.Fprintln(os.Stderr, "error: copying templates:", err)
		return 1
	}

	ctx := context.Background()

	if err := xexec.Run(ctx, dest, "go", "mod", "init", slug); err != nil {
		fmt.Fprintln(os.Stderr, "error: go mod init failed:", err)
		return 1
	}
	if err := pinGonextModule(dest); err != nil {
		fmt.Fprintln(os.Stderr, "error: pinning the gonext module failed:", err)
		return 1
	}
	if err := xexec.Run(ctx, dest, "go", "get", "-tool", "github.com/google/wire/cmd/wire"); err != nil {
		fmt.Fprintln(os.Stderr, "error: go get -tool wire failed:", err)
		return 1
	}
	if err := xexec.Run(ctx, dest, "go", "mod", "tidy"); err != nil {
		fmt.Fprintln(os.Stderr, "error: go mod tidy failed:", err)
		return 1
	}

	frontendDir := filepath.Join(dest, "frontend")
	if err := xexec.Run(ctx, frontendDir, "pnpm", "install"); err != nil {
		fmt.Fprintln(os.Stderr, "error: pnpm install failed:", err)
		return 1
	}

	if err := copyLocalConfig(dest); err != nil {
		fmt.Fprintln(os.Stderr, "error: copying mise.local.toml:", err)
		return 1
	}

	fmt.Println()
	fmt.Println("Created", dest)
	fmt.Println("Next steps:")
	fmt.Println("  cd", dest)
	fmt.Println("  mise install")
	fmt.Println("  make db-up && make migrate")
	fmt.Println("  make hooks-install")

	return 0
}

// parseInitArgs splits args into the positional name/path and the
// --agents=<list> flag, which may appear anywhere among them.
func parseInitArgs(args []string) (name, path, agents string, agentsSet bool, err error) {
	var positionals []string
	for _, arg := range args {
		if rest, ok := strings.CutPrefix(arg, "--agents="); ok {
			agents = rest
			agentsSet = true
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return "", "", "", false, fmt.Errorf("unknown flag %q", arg)
		}
		positionals = append(positionals, arg)
	}
	if len(positionals) > 0 {
		name = positionals[0]
	}
	if len(positionals) > 1 {
		path = positionals[1]
	}
	return name, path, agents, agentsSet, nil
}

// resolveSlug obtains and validates the project slug from name,
// prompting interactively when name is empty and stdin is a TTY.
func resolveSlug(name string) (string, error) {
	slug, err := scaffold.ResolveSlugArg(name, scaffold.IsTTY())
	switch {
	case err == nil:
		if valErr := scaffold.ValidateSlug(slug); valErr != nil {
			return "", valErr
		}
		return slug, nil
	case errors.Is(err, scaffold.ErrNameRequired):
		return "", err
	default:
		// errNeedsPrompt: fall through to the interactive prompt,
		// which re-prompts on invalid input via its own validator.
		return scaffold.PromptSlug()
	}
}

// copyLocalConfig copies dest/mise.local.toml.example to dest/mise.local.toml
// verbatim, owner-only since it is where a developer keeps secrets.
func copyLocalConfig(dest string) error {
	data, err := os.ReadFile(filepath.Join(dest, localConfigExample))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dest, localConfig), data, 0o600)
}

// pinGonextModule ties the project to the gonext this CLI was built from
// before `go mod tidy` gets a chance to resolve something newer.
func pinGonextModule(dest string) error {
	editArgs, err := scaffold.ModuleEdit(dest)
	if err != nil {
		return err
	}
	return xexec.Run(context.Background(), dest, "go", append([]string{"mod", "edit"}, editArgs...)...)
}
