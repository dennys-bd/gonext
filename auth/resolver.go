package auth

import (
	"context"
	"errors"
)

// Resolver turns a credential into an Identity. Implementations are
// providers: the homegrown session store a generated project ships
// with, or a third party. It receives an opaque token because the
// caller owns transport — a provider never learns whether the
// credential arrived in a cookie or a header.
type Resolver interface {
	Resolve(ctx context.Context, token string) (Identity, error)
}

// ErrUnauthenticated marks a credential that is genuinely not valid:
// absent, malformed, expired, or revoked. A Resolver must return it
// (or wrap it) for those cases and must NOT return it for
// infrastructure failures — a database outage has to surface as a
// 500, not as "your session expired".
var ErrUnauthenticated = errors.New("auth: credential is not valid")
