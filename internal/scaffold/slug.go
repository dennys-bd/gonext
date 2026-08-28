package scaffold

import (
	"errors"
	"fmt"
	"regexp"
)

// ErrInvalidSlug is the sentinel error wrapped by ValidateSlug when a
// slug fails validation. Check with errors.Is.
var ErrInvalidSlug = errors.New("invalid slug")

var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ValidateSlug reports whether s is safe to use as a Go import path
// segment: lowercase letters, digits, and hyphens only, and it must
// start with a lowercase letter.
func ValidateSlug(s string) error {
	if !slugPattern.MatchString(s) {
		return fmt.Errorf("%w: %q must contain only lowercase letters, digits, and hyphens, and start with a lowercase letter", ErrInvalidSlug, s)
	}
	return nil
}
