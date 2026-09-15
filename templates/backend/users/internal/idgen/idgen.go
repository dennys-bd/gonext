// Package idgen mints the users domain's random identifiers: user ids,
// one-shot token values, and session tokens. It exists separately so both
// application and infrastructure, which must not import each other, can share it.
package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// entropyBytes is the width of every generated id — 256 bits, far past guessing
// range, is what lets session tokens use a fast hash instead of a password-grade one.
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
