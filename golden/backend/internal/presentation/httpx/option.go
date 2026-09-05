package httpx

import "github.com/danielgtaylor/huma/v2"

// Option adjusts the huma.Operation a verb function is building. The
// set below covers what the registered routes actually declare;
// further options get added when a route needs one, not in
// anticipation.
type Option func(*huma.Operation)

// Summary sets the operation's one-line summary.
func Summary(s string) Option {
	return func(op *huma.Operation) { op.Summary = s }
}

// Description sets the operation's longer prose description.
func Description(s string) Option {
	return func(op *huma.Operation) { op.Description = s }
}

// Status sets the status a successful response carries, for the
// routes where 200 is wrong (201 on create, 202 on accept).
func Status(code int) Option {
	return func(op *huma.Operation) { op.DefaultStatus = code }
}

// Secured declares what the operation requires of the caller. It takes
// the output of auth.Required, auth.RequireRole and
// auth.RequirePermission unchanged; the auth middleware enforces it
// before the handler runs.
func Secured(security []map[string][]string) Option {
	return func(op *huma.Operation) { op.Security = security }
}

// Deprecated marks the operation deprecated in the generated OpenAPI
// document and the clients built from it.
func Deprecated() Option {
	return func(op *huma.Operation) { op.Deprecated = true }
}
