package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dennys-bd/gonext/internal/openapi"
)

func TestCamelCase(t *testing.T) {
	tests := map[string]string{
		"get-current-user": "getCurrentUser",
		"get-stub":         "getStub",
		"logout-user":      "logoutUser",
	}
	for in, want := range tests {
		if got := camelCase(in); got != want {
			t.Errorf("camelCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPascalCase(t *testing.T) {
	tests := map[string]string{
		"get-current-user": "GetCurrentUser",
		"create-stub":      "CreateStub",
	}
	for in, want := range tests {
		if got := pascalCase(in); got != want {
			t.Errorf("pascalCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHumanize(t *testing.T) {
	tests := map[string]string{
		"createdAt": "Created at",
		"get-stub":  "Get stub",
		"name":      "Name",
	}
	for in, want := range tests {
		if got := humanize(in); got != want {
			t.Errorf("humanize(%q) = %q, want %q", in, got, want)
		}
	}
}

// fixtureDoc loads internal/openapi/testdata/openapi.yaml.
func fixtureDoc(t *testing.T) *openapi.Document {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("creating docs/: %v", err)
	}
	data, err := os.ReadFile(filepath.Join("..", "openapi", "testdata", "openapi.yaml"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, openapi.DocumentPath), data, 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	doc, err := openapi.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return doc
}

func mustOperation(t *testing.T, doc *openapi.Document, id string) *openapi.Operation {
	t.Helper()
	op, err := doc.Operation(id)
	if err != nil {
		t.Fatalf("Operation(%q): %v", id, err)
	}
	return op
}

func TestBuildPageModel_Errors(t *testing.T) {
	doc := fixtureDoc(t)

	tests := []struct {
		name    string
		opID    string
		route   string
		wantErr string
	}{
		{
			name:    "unsupported method",
			opID:    "head-healthz",
			route:   "healthz",
			wantErr: `unsupported method HEAD for operation "head-healthz"`,
		},
		{
			name:    "path param not satisfiable",
			opID:    "get-stub",
			route:   "stubs/new",
			wantErr: `operation "get-stub" needs path parameter "id": add an [id] segment to the route`,
		},
		{
			name:    "optional catch-all cannot satisfy",
			opID:    "get-stub",
			route:   "stubs/[[...id]]",
			wantErr: `operation "get-stub" needs path parameter "id": add an [id] segment to the route`,
		},
		{
			name:    "required query parameter",
			opID:    "list-widgets",
			route:   "widgets",
			wantErr: `unsupported: operation "list-widgets" has a required query parameter "page"`,
		},
		{
			name:    "no application/json body",
			opID:    "upload-widget",
			route:   "widgets/[slug]",
			wantErr: `unsupported: operation "upload-widget" has no application/json request body`,
		},
		{
			name:    "unresolvable $ref",
			opID:    "rename-widget",
			route:   "widgets/[slug]",
			wantErr: `unsupported $ref "#/definitions/RenameWidgetInputBody" in docs/openapi.yaml`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := mustOperation(t, doc, tt.opID)
			r, err := parseRoute(tt.route)
			if err != nil {
				t.Fatalf("parseRoute(%q): %v", tt.route, err)
			}
			_, err = buildPageModel(doc, op, r)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("buildPageModel: err = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestBuildPageModel_SchemaMapping(t *testing.T) {
	doc := fixtureDoc(t)
	op := mustOperation(t, doc, "update-widget")
	r, err := parseRoute("widgets/[slug]")
	if err != nil {
		t.Fatalf("parseRoute: %v", err)
	}

	m, err := buildPageModel(doc, op, r)
	if err != nil {
		t.Fatalf("buildPageModel: %v", err)
	}

	inputByName := map[string]pageInput{}
	for _, in := range m.Inputs {
		inputByName[in.Name] = in
	}

	tests := []struct {
		name         string
		wantComp     string
		wantType     string
		wantCoercion string
		wantTodo     bool
	}{
		{"count", "NumberInput", "", `Number(formData.get("count"))`, false},
		{"email", "TextInput", "email", `String(formData.get("email") ?? "")`, false},
		{"enabled", "Checkbox", "", `formData.get("enabled") === "on"`, false},
		{"password", "PasswordInput", "", `String(formData.get("password") ?? "")`, false},
		{"note", "TextInput", "", `String(formData.get("note") ?? "") || undefined`, false},
		{"price", "NumberInput", "", `formData.get("price") ? Number(formData.get("price")) : undefined`, false},
		{"tags", "", "", "[]", true},
	}
	for _, tt := range tests {
		in, ok := inputByName[tt.name]
		if !ok {
			t.Fatalf("no input built for %q", tt.name)
		}
		if in.Todo != tt.wantTodo {
			t.Errorf("%s: Todo = %v, want %v", tt.name, in.Todo, tt.wantTodo)
		}
		if in.Component != tt.wantComp {
			t.Errorf("%s: Component = %q, want %q", tt.name, in.Component, tt.wantComp)
		}
		if in.InputType != tt.wantType {
			t.Errorf("%s: InputType = %q, want %q", tt.name, in.InputType, tt.wantType)
		}
		if in.Coercion != tt.wantCoercion {
			t.Errorf("%s: Coercion = %q, want %q", tt.name, in.Coercion, tt.wantCoercion)
		}
	}
	if _, ok := inputByName["$schema"]; ok {
		t.Errorf("$schema should never become an input")
	}
	if _, ok := inputByName["meta"]; ok {
		t.Errorf("meta is a freeform object (no properties, no $ref): it must be omitted from the body literal entirely, not emitted as a TODO input")
	}

	fieldByName := map[string]pageField{}
	for _, f := range m.Fields {
		fieldByName[f.Name] = f
	}
	if f := fieldByName["tags"]; !f.Todo {
		t.Errorf("tags field: Todo = false, want true (array response property)")
	}
	if f := fieldByName["enabled"]; !f.IsBoolean {
		t.Errorf("enabled field: IsBoolean = false, want true")
	}
	if f := fieldByName["name"]; !f.IsString {
		t.Errorf("name field: IsString = false, want true")
	}

	var warnCount int
	for _, w := range m.Warnings {
		if w == `skipped non-primitive property "meta"` || w == `skipped non-primitive property "tags"` {
			warnCount++
		}
	}
	if warnCount < 2 {
		t.Errorf("Warnings = %v, want entries for meta and tags", m.Warnings)
	}
}

func TestBuildPageModel_ResponseKindNone(t *testing.T) {
	doc := fixtureDoc(t)
	op := mustOperation(t, doc, "logout-user")
	r, err := parseRoute("logout")
	if err != nil {
		t.Fatalf("parseRoute: %v", err)
	}

	m, err := buildPageModel(doc, op, r)
	if err != nil {
		t.Fatalf("buildPageModel: %v", err)
	}
	if m.ResponseKind != "none" {
		t.Errorf("ResponseKind = %q, want none", m.ResponseKind)
	}
	if m.HasData {
		t.Errorf("HasData = true, want false for a 204 response")
	}
}

// TestBuildPageModel_GetWithNoContent covers the GET-with-no-content
// variant, which is not one of the fixture operations (per the plan's
// risk note, exercised here rather than with a sixth golden case).
func TestBuildPageModel_GetWithNoContent(t *testing.T) {
	doc := &openapi.Document{}
	op := &openapi.Operation{
		OperationID: "ping",
		Method:      "GET",
		Path:        "/ping",
		Responses: map[string]openapi.Response{
			"204": {},
		},
	}
	r, err := parseRoute("ping")
	if err != nil {
		t.Fatalf("parseRoute: %v", err)
	}

	m, err := buildPageModel(doc, op, r)
	if err != nil {
		t.Fatalf("buildPageModel: %v", err)
	}
	if m.ResponseKind != "none" {
		t.Errorf("ResponseKind = %q, want none", m.ResponseKind)
	}
	if m.HasData {
		t.Errorf("HasData = true, want false")
	}
}

// TestBuildPageModel_CatchAllTestLiterals pins CRITICAL 1: the literal
// passed to the action under test must match the catch-all param's own
// TS type (string[]), while the mocked SDK call assertion keeps the
// joined scalar the action itself produces.
func TestBuildPageModel_CatchAllTestLiterals(t *testing.T) {
	doc := fixtureDoc(t)
	op := mustOperation(t, doc, "update-widget")
	r, err := parseRoute("widgets/[...slug]")
	if err != nil {
		t.Fatalf("parseRoute: %v", err)
	}

	m, err := buildPageModel(doc, op, r)
	if err != nil {
		t.Fatalf("buildPageModel: %v", err)
	}

	wantPathArg := `{ slug: ["example"] }`
	if m.TestPathArg != wantPathArg {
		t.Errorf("TestPathArg = %q, want %q", m.TestPathArg, wantPathArg)
	}
	if !strings.Contains(m.TestCallWith, `path: { slug: "example" }`) {
		t.Errorf("TestCallWith = %q, want it to contain %q", m.TestCallWith, `path: { slug: "example" }`)
	}
}

// TestBuildPageModel_NilPropertySchema covers a YAML `properties: { foo: }`
// entry, which decodes to a nil *Schema: it must be treated as
// non-primitive (skipped + warned), never dereferenced.
func TestBuildPageModel_NilPropertySchema(t *testing.T) {
	doc := &openapi.Document{}
	op := &openapi.Operation{
		OperationID: "do-thing",
		Method:      "POST",
		Path:        "/do-thing",
		RequestBody: &openapi.RequestBody{
			Content: map[string]openapi.MediaType{
				"application/json": {
					Schema: &openapi.Schema{
						Type: "object",
						Properties: map[string]*openapi.Schema{
							"name":  {Type: "string"},
							"weird": nil,
						},
					},
				},
			},
		},
		Responses: map[string]openapi.Response{"204": {}},
	}
	r, err := parseRoute("do-thing")
	if err != nil {
		t.Fatalf("parseRoute: %v", err)
	}

	m, err := buildPageModel(doc, op, r)
	if err != nil {
		t.Fatalf("buildPageModel: %v", err)
	}

	var warned bool
	for _, w := range m.Warnings {
		if w == `skipped non-primitive property "weird"` {
			warned = true
		}
	}
	if !warned {
		t.Errorf("Warnings = %v, want an entry for the nil property %q", m.Warnings, "weird")
	}
	for _, in := range m.Inputs {
		if in.Name == "name" && in.Todo {
			t.Errorf("name: Todo = true, want false (a nil sibling property must not affect it)")
		}
	}
}

// TestBuildPageModel_UnsafeSummaryFallsBack covers the MEDIUM finding
// (JSX-unsafe characters) and the CRITICAL finding's Title half: a
// summary containing a quote or backslash must not become Title
// verbatim either, since Title also lands in a JS string literal in
// the *.test.tsx templates.
func TestBuildPageModel_UnsafeSummaryFallsBack(t *testing.T) {
	tests := []struct {
		name    string
		summary string
	}{
		{"JSX-unsafe characters", "Do <the> {thing}"},
		{"quote breaks out of a JS string literal", `Do the "thing"`},
		{"backslash breaks out of a JS string literal", `Do the \thing`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &openapi.Document{}
			op := &openapi.Operation{
				OperationID: "do-thing",
				Method:      "GET",
				Path:        "/do-thing",
				Summary:     tt.summary,
				Responses:   map[string]openapi.Response{"204": {}},
			}
			r, err := parseRoute("do-thing")
			if err != nil {
				t.Fatalf("parseRoute: %v", err)
			}

			m, err := buildPageModel(doc, op, r)
			if err != nil {
				t.Fatalf("buildPageModel: %v", err)
			}

			wantTitle := humanize(op.OperationID)
			if m.Title != wantTitle {
				t.Errorf("Title = %q, want the humanized operationId %q", m.Title, wantTitle)
			}
			var warned bool
			for _, w := range m.Warnings {
				if strings.Contains(w, "JSX-unsafe") {
					warned = true
				}
			}
			if !warned {
				t.Errorf("Warnings = %v, want a JSX-unsafe summary warning", m.Warnings)
			}
		})
	}
}

// TestBuildPageModel_InvalidPropertyName covers the CRITICAL finding:
// a request/response property name that is not a valid JS identifier
// must never reach the rendered TS/TSX, since it lands unescaped in
// bare object-key position, member access, JSX text and JSX attribute
// values across the templates.
func TestBuildPageModel_InvalidPropertyName(t *testing.T) {
	tests := []struct {
		name     string
		propName string
	}{
		{"kebab-case", "first-name"},
		{"template/expression injection payload", "a\"};</script><script>alert(`x`)</script>{\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &openapi.Operation{
				OperationID: "do-thing",
				Method:      "POST",
				Path:        "/do-thing",
				RequestBody: &openapi.RequestBody{
					Content: map[string]openapi.MediaType{
						"application/json": {
							Schema: &openapi.Schema{
								Type: "object",
								Properties: map[string]*openapi.Schema{
									"name":      {Type: "string"},
									tt.propName: {Type: "string"},
								},
							},
						},
					},
				},
				Responses: map[string]openapi.Response{
					"200": {
						Content: map[string]openapi.MediaType{
							"application/json": {
								Schema: &openapi.Schema{
									Type: "object",
									Properties: map[string]*openapi.Schema{
										"name":      {Type: "string"},
										tt.propName: {Type: "string"},
									},
								},
							},
						},
					},
				},
			}
			r, err := parseRoute("do-thing")
			if err != nil {
				t.Fatalf("parseRoute: %v", err)
			}

			m, err := buildPageModel(&openapi.Document{}, op, r)
			if err != nil {
				t.Fatalf("buildPageModel: %v", err)
			}

			for _, in := range m.Inputs {
				if in.Name == tt.propName {
					t.Errorf("Inputs contains hostile property %q, want it skipped entirely", tt.propName)
				}
			}
			for _, f := range m.Fields {
				if f.Name == tt.propName {
					t.Errorf("Fields contains hostile property %q, want it skipped entirely", tt.propName)
				}
			}

			wantWarning := fmt.Sprintf("skipped property %q: not a valid identifier", tt.propName)
			var warned bool
			for _, w := range m.Warnings {
				if w == wantWarning {
					warned = true
				}
			}
			if !warned {
				t.Errorf("Warnings = %v, want %q", m.Warnings, wantWarning)
			}
		})
	}
}

func TestBuildPageModel_RejectsUnsafeOperationIDAndTag(t *testing.T) {
	tests := []struct {
		name    string
		opID    string
		tags    []string
		wantErr string
	}{
		{"path traversal in operationId", "../../../tmp/pwned", nil, `unsupported: operation id "../../../tmp/pwned" is not an identifier`},
		{"quote in operationId", `foo"; import("x"); //`, nil, `unsupported: operation id "foo\"; import(\"x\"); //" is not an identifier`},
		{"leading digit in operationId", "1get", nil, `unsupported: operation id "1get" is not an identifier`},
		{"space in tag", "get-stub", []string{"My Tag"}, `unsupported: operation "get-stub" has tag "My Tag"; tags must be identifiers`},
		{"brace in tag", "get-stub", []string{"a}b"}, `unsupported: operation "get-stub" has tag "a}b"; tags must be identifiers`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &openapi.Operation{
				OperationID: tt.opID,
				Method:      "GET",
				Path:        "/x",
				Tags:        tt.tags,
				Responses:   map[string]openapi.Response{"204": {}},
			}
			r, err := parseRoute("x")
			if err != nil {
				t.Fatalf("parseRoute: %v", err)
			}

			_, err = buildPageModel(&openapi.Document{}, op, r)

			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("buildPageModel error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
