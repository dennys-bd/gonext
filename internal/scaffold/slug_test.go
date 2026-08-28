package scaffold

import (
	"errors"
	"testing"
)

func TestValidateSlug_Valid(t *testing.T) {
	valid := []string{"app", "my-app", "app2", "a", "a-b-c-1-2-3"}
	for _, s := range valid {
		if err := ValidateSlug(s); err != nil {
			t.Errorf("ValidateSlug(%q): unexpected error: %v", s, err)
		}
	}
}

func TestValidateSlug_Empty(t *testing.T) {
	if err := ValidateSlug(""); !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("ValidateSlug(\"\"): expected ErrInvalidSlug, got %v", err)
	}
}

func TestValidateSlug_Uppercase(t *testing.T) {
	if err := ValidateSlug("MyApp"); !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("ValidateSlug(\"MyApp\"): expected ErrInvalidSlug, got %v", err)
	}
}

func TestValidateSlug_Spaces(t *testing.T) {
	if err := ValidateSlug("my app"); !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("ValidateSlug(\"my app\"): expected ErrInvalidSlug, got %v", err)
	}
}

func TestValidateSlug_Slash(t *testing.T) {
	if err := ValidateSlug("my/app"); !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("ValidateSlug(\"my/app\"): expected ErrInvalidSlug, got %v", err)
	}
}

func TestValidateSlug_LeadingDigit(t *testing.T) {
	if err := ValidateSlug("1app"); !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("ValidateSlug(\"1app\"): expected ErrInvalidSlug, got %v", err)
	}
}

func TestValidateSlug_LeadingHyphen(t *testing.T) {
	if err := ValidateSlug("-app"); !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("ValidateSlug(\"-app\"): expected ErrInvalidSlug, got %v", err)
	}
}
