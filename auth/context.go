package auth

import "context"

type ctxKey struct{}

// ContextKey is the context key an Identity is stored under.
//
// It is exported because the middleware that injects the identity
// lives in a different module and goes through its own transport's
// context helper (for Huma, huma.WithValue), which writes to the
// underlying context.Context that IdentityFrom then reads. The type
// assertion in IdentityFrom is what keeps an exported key from being
// a hole: a value of any other type reads back as absent.
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
//
// That is only reachable from a handler whose operation declared
// Optional() (or declared nothing) and then asked for a guaranteed
// identity — a programming error. Panicking is deliberate: an HTTP
// server's recovery middleware turns it into a 500 and a stack trace,
// which is louder and safer than silently handing back a zero-value
// user ID that a query would then treat as a real one.
func MustIdentity(ctx context.Context) Identity {
	id, ok := IdentityFrom(ctx)
	if !ok {
		panic("auth: no identity in context; declare auth.Required() on the operation, or use IdentityFrom")
	}
	return id
}
