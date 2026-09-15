package openapi

import (
	"os"
	"path/filepath"
	"testing"
)

// testdataFixture copies internal/openapi/testdata/openapi.yaml into a
// fresh project root's docs/ directory.
func testdataFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("creating docs/: %v", err)
	}
	data, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, DocumentPath), data, 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	return root
}

func TestLoad_ParsesFixture(t *testing.T) {
	root := testdataFixture(t)

	doc, err := Load(root)
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}

	op, err := doc.Operation("get-stub")
	if err != nil {
		t.Fatalf("Operation(get-stub): unexpected error: %v", err)
	}
	if op.Method != "GET" {
		t.Errorf("get-stub: Method = %q, want GET", op.Method)
	}
	if op.Path != "/stubs/{id}" {
		t.Errorf("get-stub: Path = %q, want /stubs/{id}", op.Path)
	}
	if len(op.Parameters) != 1 || op.Parameters[0].Name != "id" {
		t.Errorf("get-stub: Parameters = %+v, want one param named id", op.Parameters)
	}

	errorModel, ok := doc.Components.Schemas["ErrorModel"]
	if !ok {
		t.Fatalf("ErrorModel schema not found")
	}
	errorsProp, ok := errorModel.Properties["errors"]
	if !ok {
		t.Fatalf("ErrorModel.errors property not found")
	}
	if errorsProp.Type != "array" {
		t.Errorf("ErrorModel.errors.Type = %q, want array", errorsProp.Type)
	}
	if !errorsProp.Nullable {
		t.Errorf("ErrorModel.errors.Nullable = false, want true")
	}

	stubOutput, ok := doc.Components.Schemas["StubOutputBody"]
	if !ok {
		t.Fatalf("StubOutputBody schema not found")
	}
	schemaProp, ok := stubOutput.Properties["$schema"]
	if !ok {
		t.Fatalf("StubOutputBody.$schema property not found")
	}
	if !schemaProp.ReadOnly {
		t.Errorf("StubOutputBody.$schema.ReadOnly = false, want true")
	}
}

func TestLoad_MissingDocument(t *testing.T) {
	root := t.TempDir()

	_, err := Load(root)
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	want := "docs/openapi.yaml not found; run `gonext openapi`"
	if err.Error() != want {
		t.Errorf("Load: err = %q, want %q", err.Error(), want)
	}
}

func TestOperation_Unknown(t *testing.T) {
	root := testdataFixture(t)
	doc, err := Load(root)
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}

	_, err = doc.Operation("get-stubb")
	if err == nil {
		t.Fatal("Operation: expected error, got nil")
	}
	want := `no operation "get-stubb" in docs/openapi.yaml`
	if err.Error() != want {
		t.Errorf("Operation: err = %q, want %q", err.Error(), want)
	}
}

func TestResolve(t *testing.T) {
	root := testdataFixture(t)
	doc, err := Load(root)
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}

	tests := []struct {
		name    string
		schema  *Schema
		wantErr string
	}{
		{name: "nil", schema: nil},
		{name: "no ref", schema: &Schema{Type: "string"}},
		{name: "component ref", schema: &Schema{Ref: "#/components/schemas/StubOutputBody"}},
		{
			name:    "unsupported ref form",
			schema:  &Schema{Ref: "#/definitions/X"},
			wantErr: `unsupported $ref "#/definitions/X" in docs/openapi.yaml`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := doc.Resolve(tt.schema)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("Resolve: err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve: unexpected error: %v", err)
			}
			if tt.schema != nil && tt.schema.Ref == "" && resolved != tt.schema {
				t.Errorf("Resolve: expected passthrough for non-ref schema")
			}
			if tt.name == "component ref" && (resolved == nil || resolved.Type != "object") {
				t.Errorf("Resolve: component ref resolved = %+v, want StubOutputBody schema", resolved)
			}
		})
	}
}
