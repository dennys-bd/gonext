// Package openapi builds the Huma API with every domain registered, over
// whatever infrastructure it is handed, so `gonext openapi` can produce the
// document with no database and no environment. Nothing in a running server imports it.
package openapi

import (
	"github.com/danielgtaylor/huma/v2"

	"golden-app/backend/example"
	"golden-app/backend/internal/presentation/api"
	"golden-app/backend/users"
)

// Spec is what Initialize returns: the API with every registration
// applied. It mirrors main's App but carries only what producing the
// document needs.
type Spec struct {
	API huma.API
}

// NewSpec assembles Spec. Its marker parameters aren't read; depending on
// them is what forces wire to run every registration before Initialize returns.
func NewSpec(
	humaAPI huma.API,
	_ api.HealthzRegistered,
	_ api.ReadyzRegistered,
	_ example.Registered,
	_ users.Registered,
) *Spec {
	return &Spec{API: humaAPI}
}
