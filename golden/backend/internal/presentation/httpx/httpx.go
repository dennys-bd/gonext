// Package httpx wraps Huma's registration so handlers receive a *Ctx instead
// of a bare context.Context. Register is the adapter; a lint rule forbids
// calling huma.Register directly outside this package.
package httpx

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dennys-bd/gonext/auth"
)

// Ctx is the request context handlers receive. It embeds
// context.Context, so *Ctx passes unchanged to services,
// repositories, and anything else taking one.
type Ctx struct {
	context.Context

	identity    auth.Identity
	hasIdentity bool
}

// NewCtx wraps ctx, lifting any identity the auth middleware injected
// into a field. Exported for tests; handlers receive a *Ctx from
// Register and never construct one.
func NewCtx(ctx context.Context) *Ctx {
	identity, ok := auth.IdentityFrom(ctx)
	return &Ctx{Context: ctx, identity: identity, hasIdentity: ok}
}

// Identity returns the authenticated identity. It panics unless the
// operation declared auth.Required(), RequireRole or RequirePermission;
// use IdentityOK when auth is optional or undeclared.
func (c *Ctx) Identity() auth.Identity {
	if !c.hasIdentity {
		panic("httpx: no identity in context; declare auth.Required() on the operation, or use IdentityOK")
	}
	return c.identity
}

// IdentityOK returns the authenticated identity, reporting whether
// one is present. This is the accessor for operations declaring
// auth.Optional().
func (c *Ctx) IdentityOK() (auth.Identity, bool) {
	return c.identity, c.hasIdentity
}

// Register registers op on api, adapting a handler that takes *Ctx to
// the signature huma.Register requires.
func Register[I, O any](api huma.API, op huma.Operation, handler func(*Ctx, *I) (*O, error)) {
	huma.Register(api, op, func(ctx context.Context, in *I) (*O, error) {
		return handler(NewCtx(ctx), in)
	})
}
