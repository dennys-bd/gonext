package generate

import (
	"strings"
	"testing"
)

func TestParseFields(t *testing.T) {
	tests := []struct {
		name    string
		tokens  []string
		want    []resourceField
		wantErr string
	}{
		{
			name:   "string field",
			tokens: []string{"title:string"},
			want:   []resourceField{{Name: "title", Pascal: "Title", Camel: "title", Column: "title", Type: fieldTypes["string"]}},
		},
		{
			name:   "snake_case casing",
			tokens: []string{"order_line:time"},
			want:   []resourceField{{Name: "order_line", Pascal: "OrderLine", Camel: "orderLine", Column: "order_line", Type: fieldTypes["time"]}},
		},
		{
			name:   "no fields",
			tokens: nil,
			want:   nil,
		},
		{name: "no colon", tokens: []string{"title"}, wantErr: `invalid field "title": use name:type`},
		{name: "empty name", tokens: []string{":string"}, wantErr: `invalid field ":string": use name:type`},
		{name: "empty type", tokens: []string{"title:"}, wantErr: `invalid field "title:": use name:type`},
		{name: "not snake_case", tokens: []string{"Title:string"}, wantErr: `invalid field name "Title": use snake_case`},
		{name: "id is implicit", tokens: []string{"id:string"}, wantErr: `field "id" is implicit`},
		{name: "created_at is implicit", tokens: []string{"created_at:time"}, wantErr: `field "created_at" is implicit`},
		{name: "updated_at is implicit", tokens: []string{"updated_at:time"}, wantErr: `field "updated_at" is implicit`},
		{name: "repeated", tokens: []string{"a:int", "a:string"}, wantErr: `field "a" repeated`},
		{name: "unknown type", tokens: []string{"x:uuid"}, wantErr: `unknown type "uuid" for field "x": use string, int, int64, float, bool or time`},
		{name: "go keyword", tokens: []string{"type:string"}, wantErr: `field "type" is a Go keyword`},
		{name: "reserved local", tokens: []string{"ctx:string"}, wantErr: `field "ctx" is reserved`},
		{name: "reserved receiver", tokens: []string{"s:string"}, wantErr: `field "s" is reserved`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFields(tt.tokens)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("parseFields: err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseFields: unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseFields: got %d fields, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseFields[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFieldTypes(t *testing.T) {
	tests := []struct {
		token string
		want  fieldType
	}{
		{"string", fieldType{Token: "string", Go: "string", Column: "text NOT NULL", HumaTags: `minLength:"1"`, GoSample: `"demo"`, GoUpdated: `"demo-updated"`, BruSample: `"demo"`, BruUpdated: `"demo-updated"`}},
		{"int", fieldType{Token: "int", Go: "int", Column: "integer NOT NULL", GoSample: "42", GoUpdated: "43", BruSample: "42", BruUpdated: "43"}},
		{"int64", fieldType{Token: "int64", Go: "int64", Column: "bigint NOT NULL", JSONFormat: "int64", GoSample: "int64(42)", GoUpdated: "int64(43)", BruSample: "42", BruUpdated: "43"}},
		{"float", fieldType{Token: "float", Go: "float64", Column: "double precision NOT NULL", GoSample: "9.5", GoUpdated: "10.5", BruSample: "9.5", BruUpdated: "10.5"}},
		{"bool", fieldType{Token: "bool", Go: "bool", Column: "boolean NOT NULL", GoSample: "true", GoUpdated: "false", BruSample: "true", BruUpdated: "false"}},
		{"time", fieldType{Token: "time", Go: "time.Time", Column: "timestamptz NOT NULL", JSONFormat: "date-time", GoSample: "time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)", GoUpdated: "time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)", BruSample: `"2026-01-02T03:04:05Z"`, BruUpdated: `"2026-02-03T04:05:06Z"`, ImportsTime: true}},
	}

	if len(fieldTypes) != len(tests) {
		t.Fatalf("fieldTypes has %d entries, want %d", len(fieldTypes), len(tests))
	}
	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			got, ok := fieldTypes[tt.token]
			if !ok {
				t.Fatalf("fieldTypes[%q] missing", tt.token)
			}
			if got != tt.want {
				t.Errorf("fieldTypes[%q] = %+v, want %+v", tt.token, got, tt.want)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := map[string]string{
		"product":  "products",
		"category": "categories",
		"day":      "days",
		"key":      "keys",
		"box":      "boxes",
		"bus":      "buses",
		"quiz":     "quizes",
		"dish":     "dishes",
		"match":    "matches",
	}
	for in, want := range tests {
		if got := pluralize(in); got != want {
			t.Errorf("pluralize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseOps(t *testing.T) {
	tests := []struct {
		name    string
		ops     []string
		want    string
		wantErr string
	}{
		{name: "nil selects all", ops: nil, want: "create,get,list,update,delete"},
		{name: "empty selects all", ops: []string{}, want: "create,get,list,update,delete"},
		{name: "canonical order", ops: []string{"get", "create"}, want: "create,get"},
		{name: "duplicates collapse", ops: []string{"delete", "delete"}, want: "delete"},
		{name: "empty entry", ops: []string{"create", "", "get"}, wantErr: `unknown operation "": use create, get, list, update or delete`},
		{name: "unknown op", ops: []string{"patch"}, wantErr: `unknown operation "patch": use create, get, list, update or delete`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOps(tt.ops)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("parseOps: err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOps: unexpected error: %v", err)
			}
			if joined := strings.Join(got, ","); joined != tt.want {
				t.Errorf("parseOps = %q, want %q", joined, tt.want)
			}
		})
	}
}

func TestSecuredOps(t *testing.T) {
	all := []string{"create", "get", "list", "update", "delete"}
	tests := []struct {
		name    string
		policy  string
		ops     []string
		want    opSet
		wantErr string
	}{
		{name: "all", policy: "all", ops: all, want: opSet{Create: true, Get: true, List: true, Update: true, Delete: true}},
		{name: "default is all", policy: "", ops: all, want: opSet{Create: true, Get: true, List: true, Update: true, Delete: true}},
		{name: "all only secures selected", policy: "all", ops: []string{"get", "list"}, want: opSet{Get: true, List: true}},
		{name: "mutations", policy: "mutations", ops: all, want: opSet{Create: true, Update: true, Delete: true}},
		{name: "mutations narrowed", policy: "mutations", ops: []string{"create", "get"}, want: opSet{Create: true}},
		{name: "public", policy: "public", ops: all, want: opSet{}},
		{name: "unknown", policy: "owner", ops: all, wantErr: `unknown auth policy "owner": use all, mutations or public`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := securedOps(tt.policy, tt.ops)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("securedOps: err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("securedOps: unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("securedOps = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestBuildResourceModel_Names(t *testing.T) {
	m, err := buildResourceModel("golden-app", ResourceSpec{Domain: "example", Name: "product", Fields: []string{"title:string", "price:int"}})
	if err != nil {
		t.Fatalf("buildResourceModel: unexpected error: %v", err)
	}
	checks := map[string][2]string{
		"Domain":       {m.Domain, "example"},
		"DomainPascal": {m.DomainPascal, "Example"},
		"Module":       {m.Module, "golden-app"},
		"Name":         {m.Name, "product"},
		"Pascal":       {m.Pascal, "Product"},
		"Camel":        {m.Camel, "product"},
		"Plural":       {m.Plural, "products"},
		"PluralPascal": {m.PluralPascal, "Products"},
		"PluralCamel":  {m.PluralCamel, "products"},
		"Table":        {m.Table, "products"},
		"Route":        {m.Route, "/products"},
		"SessionVar":   {m.SessionVar, "exampleSessionCookie"},
		"IDVar":        {m.IDVar, "productId"},
		"Human":        {m.Human, "product"},
		"PluralHuman":  {m.PluralHuman, "products"},
		"OpID.create":  {m.OpID["create"], "create-product"},
		"OpID.get":     {m.OpID["get"], "get-product"},
		"OpID.list":    {m.OpID["list"], "list-products"},
		"OpID.update":  {m.OpID["update"], "update-product"},
		"OpID.delete":  {m.OpID["delete"], "delete-product"},
	}
	for field, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s = %q, want %q", field, c[0], c[1])
		}
	}
	if !m.HasFields || m.ImportsTime || !m.AnySecured {
		t.Errorf("flags: HasFields=%v ImportsTime=%v AnySecured=%v, want true false true", m.HasFields, m.ImportsTime, m.AnySecured)
	}
	if m.Op != (opSet{Create: true, Get: true, List: true, Update: true, Delete: true}) || m.Op != m.Secured {
		t.Errorf("Op = %+v, Secured = %+v, want all five in both", m.Op, m.Secured)
	}
	if len(m.Fields) != 2 || m.Fields[0].Pascal != "Title" || m.Fields[1].Pascal != "Price" {
		t.Errorf("Fields = %+v, want Title, Price", m.Fields)
	}
}

func TestBuildResourceModel_Derivations(t *testing.T) {
	m, err := buildResourceModel("golden-app", ResourceSpec{
		Domain: "billing", Name: "order_line", Plural: "lines", Auth: "public",
		Fields: []string{"due:time"}, Ops: []string{"list", "get"},
	})
	if err != nil {
		t.Fatalf("buildResourceModel: unexpected error: %v", err)
	}
	checks := map[string][2]string{
		"DomainPascal": {m.DomainPascal, "Billing"},
		"Pascal":       {m.Pascal, "OrderLine"},
		"Camel":        {m.Camel, "orderLine"},
		"Plural":       {m.Plural, "lines"},
		"PluralPascal": {m.PluralPascal, "Lines"},
		"Route":        {m.Route, "/lines"},
		"SessionVar":   {m.SessionVar, "billingSessionCookie"},
		"IDVar":        {m.IDVar, "orderLineId"},
		"Human":        {m.Human, "order line"},
		"OpID.get":     {m.OpID["get"], "get-order-line"},
		"OpID.list":    {m.OpID["list"], "list-lines"},
	}
	for field, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s = %q, want %q", field, c[0], c[1])
		}
	}
	if !m.ImportsTime || m.AnySecured {
		t.Errorf("flags: ImportsTime=%v AnySecured=%v, want true false", m.ImportsTime, m.AnySecured)
	}
	if m.Op != (opSet{Get: true, List: true}) || m.Secured != (opSet{}) {
		t.Errorf("Op = %+v, Secured = %+v", m.Op, m.Secured)
	}
}

func TestBuildResourceModel_Errors(t *testing.T) {
	tests := []struct {
		name    string
		spec    ResourceSpec
		wantErr string
	}{
		{"bad name", ResourceSpec{Domain: "example", Name: "Product"}, `invalid resource name "Product": use snake_case`},
		{"bad field", ResourceSpec{Domain: "example", Name: "product", Fields: []string{"title"}}, `invalid field "title": use name:type`},
		{"field named after the resource", ResourceSpec{Domain: "example", Name: "product", Fields: []string{"product:string"}}, `field "product" is reserved`},
		{"bad plural", ResourceSpec{Domain: "example", Name: "cat", Plural: "Cats"}, `invalid plural "Cats": use snake_case`},
		{"bad op", ResourceSpec{Domain: "example", Name: "product", Ops: []string{"patch"}}, `unknown operation "patch": use create, get, list, update or delete`},
		{"bad auth", ResourceSpec{Domain: "example", Name: "product", Auth: "owner"}, `unknown auth policy "owner": use all, mutations or public`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildResourceModel("golden-app", tt.spec)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("buildResourceModel: err = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
