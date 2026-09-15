package scaffold

import (
	"errors"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
)

// ErrNameRequired is returned when name is empty and stdin is not a
// TTY, so there is no way to collect it interactively.
var ErrNameRequired = errors.New("project name is required")

// errNeedsPrompt signals that name was empty but stdin is a TTY, so
// the caller should fall back to the interactive prompt.
var errNeedsPrompt = errors.New("prompt required")

// ResolveSlugArg returns name unchanged if non-empty; otherwise
// errNeedsPrompt when isTTY (caller should prompt via PromptSlug), or
// ErrNameRequired when not.
func ResolveSlugArg(name string, isTTY bool) (string, error) {
	if name != "" {
		return name, nil
	}
	if isTTY {
		return "", errNeedsPrompt
	}
	return "", ErrNameRequired
}

// IsTTY reports whether stdin is attached to a terminal.
func IsTTY() bool {
	return isatty.IsTerminal(os.Stdin.Fd())
}

// PromptSlug interactively asks for a project name, re-prompting
// until ValidateSlug accepts it.
func PromptSlug() (string, error) {
	var name string
	field := huh.NewInput().
		Title("Project name?").
		Value(&name).
		Validate(func(s string) error {
			return ValidateSlug(s)
		})
	if err := huh.NewForm(huh.NewGroup(field)).Run(); err != nil {
		return "", err
	}
	return name, nil
}

// PromptAgents interactively asks which agent tool files to
// scaffold. Nothing is pre-selected; Codex is listed only so the
// developer sees it is already covered by AGENTS.md.
func PromptAgents() ([]string, error) {
	var picked []string
	options := make([]huh.Option[string], 0, len(AgentNames()))
	for _, name := range AgentNames() {
		label := name
		if name == "codex" {
			label = "codex (reads AGENTS.md — already covered)"
		}
		options = append(options, huh.NewOption(label, name))
	}
	field := huh.NewMultiSelect[string]().
		Title("Agent tooling?").
		Options(options...).
		Value(&picked)
	if err := huh.NewForm(huh.NewGroup(field)).Run(); err != nil {
		return nil, err
	}
	return picked, nil
}
