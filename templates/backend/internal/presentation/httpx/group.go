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
// operation with tag and logging errors withheld from clients.
// It panics if logger is nil.
func NewGroup(api huma.API, prefix, tag string, logger *slog.Logger) *Group {
	if logger == nil {
		panic("httpx: NewGroup requires a non-nil logger; it records the errors withheld from clients")
	}
	return &Group{api: api, prefix: prefix, tag: tag, logger: logger}
}

// Errors declares how the group's domain sentinels map to statuses,
// checked in declaration order, and returns the group so construction
// reads as one expression.
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

// path composes the group's prefix with a route's suffix, verbatim: pass
// "" for the collection root, not "/", which composes to a different route.
func (g *Group) path(suffix string) string {
	return g.prefix + suffix
}

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

// translate applies the group's error policy (see "Errors" in AGENTS.md):
// a declared sentinel wins first, then a status error passes through,
// then anything else is logged and replaced with a flat 500.
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
