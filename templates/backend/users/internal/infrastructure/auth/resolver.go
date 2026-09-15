// Package auth adapts the users track's session issuer to gonext's published
// auth.Resolver contract — the default identity provider, swapped by providing
// a different auth.Resolver in wire.go.
package auth

import (
	"context"
	"errors"
	"fmt"

	gonextauth "github.com/dennys-bd/gonext/auth"

	"[PROJECT-NAME]/backend/users/domain"
)

// resolver satisfies gonextauth.Resolver on top of a SessionIssuer.
type resolver struct {
	issuer domain.SessionIssuer
}

// NewResolver adapts issuer to the published Resolver contract.
func NewResolver(issuer domain.SessionIssuer) gonextauth.Resolver {
	return resolver{issuer: issuer}
}

// Resolve validates token and translates an invalid session to
// gonextauth.ErrUnauthenticated; any other error passes through so the
// middleware can tell an expired session from a database that is down.
func (r resolver) Resolve(ctx context.Context, token string) (gonextauth.Identity, error) {
	identity, _, err := r.issuer.Validate(ctx, token)
	if err != nil {
		if errors.Is(err, domain.ErrSessionInvalid) {
			return gonextauth.Identity{}, gonextauth.ErrUnauthenticated
		}
		return gonextauth.Identity{}, fmt.Errorf("validating session: %w", err)
	}
	return identity, nil
}
