package httpx_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"

	"[PROJECT-NAME]/backend/internal/presentation/httpx"
)

// Each option must set exactly the huma.Operation field it names and
// nothing else — the group builds the operation, the options are the
// only way a route says anything about its own.
func TestOptions_SetTheirOperationField(t *testing.T) {
	security := []map[string][]string{{"cookie": {}}}

	tests := []struct {
		name string
		opt  httpx.Option
		want huma.Operation
	}{
		{"Summary", httpx.Summary("Create a stub"), huma.Operation{Summary: "Create a stub"}},
		{"Description", httpx.Description("Longer prose."), huma.Operation{Description: "Longer prose."}},
		{"Status", httpx.Status(http.StatusCreated), huma.Operation{DefaultStatus: http.StatusCreated}},
		{"Secured", httpx.Secured(security), huma.Operation{Security: security}},
		{"Deprecated", httpx.Deprecated(), huma.Operation{Deprecated: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var op huma.Operation
			tt.opt(&op)

			if !reflect.DeepEqual(op, tt.want) {
				t.Errorf("option produced %+v, want %+v", op, tt.want)
			}
		})
	}
}

// Options reach the operation the group actually registers, and are
// applied after the group's own fields so a route can still be read
// off the generated OpenAPI document.
func TestOptions_ApplyToTheRegisteredOperation(t *testing.T) {
	_, api := humatest.New(t)
	g := httpx.NewGroup(api, "/stubs", "Example", discardLogger())

	httpx.Get(g, "", "list-stubs", okHandler,
		httpx.Summary("List stubs"),
		httpx.Description("Every stub."),
		httpx.Status(http.StatusAccepted),
		httpx.Secured([]map[string][]string{{"cookie": {}}}),
		httpx.Deprecated())

	op := api.OpenAPI().Paths["/stubs"].Get
	if op.Summary != "List stubs" {
		t.Errorf("Summary = %q, want %q", op.Summary, "List stubs")
	}
	if op.Description != "Every stub." {
		t.Errorf("Description = %q, want %q", op.Description, "Every stub.")
	}
	if op.DefaultStatus != http.StatusAccepted {
		t.Errorf("DefaultStatus = %d, want %d", op.DefaultStatus, http.StatusAccepted)
	}
	if len(op.Security) != 1 {
		t.Errorf("Security = %v, want one requirement", op.Security)
	}
	if !op.Deprecated {
		t.Error("Deprecated = false, want true")
	}
	// The group's own fields survive the options.
	if len(op.Tags) != 1 || op.Tags[0] != "Example" {
		t.Errorf("Tags = %v, want exactly [Example]", op.Tags)
	}
}
