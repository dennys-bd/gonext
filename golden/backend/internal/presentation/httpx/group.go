package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// Group carries the path prefix, OpenAPI tag, and error policy shared
// by one track's operations, so each of them states only what is
// actually its own: its verb, its path suffix, and its operation ID.
type Group struct {
	api    huma.API
	prefix string
	tag    string
	errors []ErrorMapping
	logger *slog.Logger
}

// NewGroup builds a group mounting on api under prefix, tagging every
// operation with tag. logger records the errors that reach the client
// as a 500 with their detail withheld.
// A nil logger is rejected here rather than tolerated, because the
// only place the group uses it is the unmapped-error path — so a nil
// would surface as a panic during an incident rather than at boot,
// which is the worst moment to discover a wiring mistake. Registration
// runs at startup, before serving, so this fails the process fast.
func NewGroup(api huma.API, prefix, tag string, logger *slog.Logger) *Group {
	if logger == nil {
		panic("httpx: NewGroup requires a non-nil logger; it records the errors withheld from clients")
	}
	return &Group{api: api, prefix: prefix, tag: tag, logger: logger}
}

// Errors declares how the group's domain sentinels map to statuses.
// It returns the group so construction reads as one expression.
//
// The mappings are an ordered slice rather than a map[error]int on
// purpose: map iteration order is randomised, so an error matching two
// declared sentinels would get a status that varied between requests.
// Declaration order is stable, and lets a specific mapping precede a
// general one.
func (g *Group) Errors(mappings ...ErrorMapping) *Group {
	g.errors = append(g.errors, mappings...)
	return g
}

// ErrorMapping pairs a domain sentinel with the status it becomes.
type ErrorMapping struct {
	Sentinel error
	Status   int
}

// Map declares one sentinel-to-status mapping.
func Map(sentinel error, status int) ErrorMapping {
	return ErrorMapping{Sentinel: sentinel, Status: status}
}

// path composes the group's prefix with a route's suffix, verbatim. A
// collection-root route passes the empty string; "/" is not a synonym
// for it, since it composes to a distinct trailing-slash route.
func (g *Group) path(suffix string) string {
	return g.prefix + suffix
}

// register is the one core every verb function delegates to: it
// composes the path, applies the group's tag and the caller's options,
// and hands huma a closure that gives the handler a *Ctx.
func register[I, O any](g *Group, method, path, operationID string, handler func(*Ctx, *I) (*O, error), opts ...Option) {
	op := huma.Operation{
		OperationID: operationID,
		Method:      method,
		Path:        g.path(path),
		Tags:        []string{g.tag},
	}
	for _, opt := range opts {
		opt(&op)
	}

	huma.Register(g.api, op, func(ctx context.Context, in *I) (*O, error) {
		out, err := handler(NewCtx(ctx), in)
		if err != nil {
			return nil, g.translate(ctx, err)
		}
		return out, nil
	})
}

// translate turns the error a handler returned into the one the client
// sees. Precedence, in this order:
//
//  1. The group's declared mappings, walked in declaration order with
//     errors.Is. A domain sentinel — wrapped for context or not —
//     becomes its declared status, and the client is sent the
//     sentinel's own message, which is written for the caller. Any
//     context a caller wrapped around it is deliberately dropped: the
//     same %w that adds "loading order 7" could add a connection
//     string, and nothing distinguishes the two here. To send the
//     client a message built at request time, declare a separate
//     sentinel or return a huma.ErrorNNN directly, which step 2 passes
//     through untouched.
//  2. Otherwise an error already carrying a status passes through, so
//     a handler can still answer directly for a case not worth
//     declaring on the group. It is the huma.StatusError itself that
//     is returned, never a wrapper around it.
//  3. Otherwise the error is logged with the request context and
//     replaced by a flat 500 carrying none of its detail.
//
// Mappings are checked first because the group's declared policy is
// its contract; the two never collide in practice, since a handler
// returning a status directly is by definition a case with no mapping.
//
// Step 3 is the leak barrier, and the reason a handler can safely
// return a bare error: huma renders a non-StatusError by putting
// err.Error() into the response body, so without it wrapped internals
// — database driver text included — would reach the client.
func (g *Group) translate(ctx context.Context, err error) error {
	for _, mapping := range g.errors {
		if errors.Is(err, mapping.Sentinel) {
			return huma.NewError(mapping.Status, mapping.Sentinel.Error())
		}
	}

	var status huma.StatusError
	if errors.As(err, &status) {
		return status
	}

	g.logger.ErrorContext(ctx, "httpx: unhandled error", "error", err)
	return huma.Error500InternalServerError("internal server error")
}

// Get registers a GET operation on the group.
func Get[I, O any](g *Group, path, operationID string, handler func(*Ctx, *I) (*O, error), opts ...Option) {
	register(g, http.MethodGet, path, operationID, handler, opts...)
}

// Post registers a POST operation on the group.
func Post[I, O any](g *Group, path, operationID string, handler func(*Ctx, *I) (*O, error), opts ...Option) {
	register(g, http.MethodPost, path, operationID, handler, opts...)
}

// Put registers a PUT operation on the group.
func Put[I, O any](g *Group, path, operationID string, handler func(*Ctx, *I) (*O, error), opts ...Option) {
	register(g, http.MethodPut, path, operationID, handler, opts...)
}

// Patch registers a PATCH operation on the group.
func Patch[I, O any](g *Group, path, operationID string, handler func(*Ctx, *I) (*O, error), opts ...Option) {
	register(g, http.MethodPatch, path, operationID, handler, opts...)
}

// Delete registers a DELETE operation on the group.
func Delete[I, O any](g *Group, path, operationID string, handler func(*Ctx, *I) (*O, error), opts ...Option) {
	register(g, http.MethodDelete, path, operationID, handler, opts...)
}
