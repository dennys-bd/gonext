package auth

import "context"

type ctxKey struct{}

// ContextKey is the context key an Identity is stored under. It is
// exported because middleware outside this module writes to it
// directly through its own transport's context helper.
var ContextKey any = ctxKey{}

// WithIdentity returns a copy of ctx carrying id. Callers holding a
// plain context.Context use this — tests, and any non-HTTP transport.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ContextKey, id)
}

// IdentityFrom returns the identity carried by ctx, reporting whether
// one was present.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(ContextKey).(Identity)
	return id, ok
}

// MustIdentity returns the identity carried by ctx and panics when
// there is none.
func MustIdentity(ctx context.Context) Identity {
	id, ok := IdentityFrom(ctx)
	if !ok {
		panic("auth: no identity in context; declare auth.Required() on the operation, or use IdentityFrom")
	}
	return id
}
