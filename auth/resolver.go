package auth

import (
	"context"
	"errors"
)

// Resolver turns a credential into an Identity. Implementations are
// providers — a generated project's own session store, or a third
// party — and receive an opaque token; the caller owns transport.
type Resolver interface {
	Resolve(ctx context.Context, token string) (Identity, error)
}

// ErrUnauthenticated marks a credential that is genuinely not valid
// (absent, malformed, expired, revoked). Resolver implementations
// must return it only for that, never for infrastructure failures.
var ErrUnauthenticated = errors.New("auth: credential is not valid")
