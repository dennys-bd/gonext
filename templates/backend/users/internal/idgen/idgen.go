// Package idgen mints the users domain's random identifiers: user
// ids, one-shot token values, and session tokens. It lives in its own
// package because application and infrastructure are siblings that
// must not import each other, yet both need the same generator.
package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// entropyBytes is the width of every generated id. 256 bits is far
// past guessing range, which is what lets session tokens be hashed
// with a fast digest instead of a password-grade one.
const entropyBytes = 32

// New returns a random 256-bit hex id. The error is surfaced rather
// than discarded: an id built from a failed read would be
// predictable, and these values are used as bearer credentials.
func New() (string, error) {
	b := make([]byte, entropyBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}
