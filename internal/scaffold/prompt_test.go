package scaffold

import (
	"errors"
	"testing"
)

func TestResolveSlugArg_UsesArgWhenGiven(t *testing.T) {
	got, err := ResolveSlugArg("my-app", true)
	if err != nil {
		t.Fatalf("ResolveSlugArg: unexpected error: %v", err)
	}
	if got != "my-app" {
		t.Errorf("ResolveSlugArg(%q, true) = %q, want %q", "my-app", got, "my-app")
	}
}

func TestResolveSlugArg_EmptyAndNotTTYIsHardError(t *testing.T) {
	_, err := ResolveSlugArg("", false)
	if !errors.Is(err, ErrNameRequired) {
		t.Fatalf("ResolveSlugArg(\"\", false): expected ErrNameRequired, got %v", err)
	}
}

func TestResolveSlugArg_EmptyAndTTYSignalsPrompt(t *testing.T) {
	_, err := ResolveSlugArg("", true)
	if !errors.Is(err, errNeedsPrompt) {
		t.Fatalf("ResolveSlugArg(\"\", true): expected errNeedsPrompt, got %v", err)
	}
}
