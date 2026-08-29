package application

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"golden-app/backend/users/domain"
)

// Argon2id cost parameters. Deliberately constants rather than
// environment settings: they are a security property of Core, not
// something a project should tune per deployment.
const (
	argonTime    uint32 = 1
	argonMemory  uint32 = 64 * 1024
	argonThreads uint8  = 4
	argonKeyLen  uint32 = 32
	argonSaltLen        = 16
)

var _ domain.PasswordHasher = (*Argon2Hasher)(nil)

// Argon2Hasher hashes passwords with Argon2id, encoding the salt and
// cost parameters alongside the digest in the standard PHC string
// format so a future parameter change stays backwards compatible with
// already-stored hashes.
type Argon2Hasher struct{}

// NewArgon2Hasher constructs an Argon2Hasher.
func NewArgon2Hasher() *Argon2Hasher {
	return &Argon2Hasher{}
}

// Hash returns a PHC-encoded Argon2id hash of password.
func (h *Argon2Hasher) Hash(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating password salt: %w", err)
	}

	digest := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(digest),
	), nil
}

// Verify reports whether password matches the PHC-encoded hash.
func (h *Argon2Hasher) Verify(password, hash string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, fmt.Errorf("decoding password hash: unrecognised format")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("decoding password hash version: %w", err)
	}
	if version != argon2.Version {
		return false, fmt.Errorf("decoding password hash: unsupported argon2 version %d", version)
	}

	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, fmt.Errorf("decoding password hash parameters: %w", err)
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("decoding password salt: %w", err)
	}
	want, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("decoding password digest: %w", err)
	}

	got := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
