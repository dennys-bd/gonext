package generate

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func productAllModel(t *testing.T) resourceModel {
	t.Helper()
	m, err := buildResourceModel("golden-app", ResourceSpec{
		Domain: "example",
		Name:   "product",
		Fields: []string{"title:string", "price:int"},
	})
	if err != nil {
		t.Fatalf("buildResourceModel: %v", err)
	}
	return m
}

func readGolden(t *testing.T, caseName, rel string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "resource", caseName, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("reading golden %s: %v", rel, err)
	}
	return data
}

func TestRenderResource_Golden(t *testing.T) {
	m := productAllModel(t)
	const migrationRel = "backend/example/migrations/0002_create_products.go"

	files, err := renderResource(m, migrationRel)
	if err != nil {
		t.Fatalf("renderResource: unexpected error: %v", err)
	}

	want := []string{
		"backend/example/domain/product.go",
		"backend/example/internal/application/product_service.go",
		"backend/example/internal/application/product_service_test.go",
		"backend/example/internal/infrastructure/memory/product_repository.go",
		"backend/example/internal/infrastructure/memory/product_repository_test.go",
		"backend/example/internal/infrastructure/postgres/product_repository.go",
		"backend/example/internal/infrastructure/postgres/product_repository_test.go",
		"backend/example/internal/presentation/product.go",
		"backend/example/internal/presentation/product_test.go",
		migrationRel,
	}
	got := make([]string, len(files))
	for i, f := range files {
		got[i] = f.rel
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("renderResource: rels = %v, want %v", got, want)
	}

	for _, f := range files {
		wantData := readGolden(t, "product-all", f.rel)
		if !bytes.Equal(f.data, wantData) {
			t.Errorf("%s: content differs\n--- want\n%s\n--- got\n%s", f.rel, wantData, f.data)
		}
	}
}

// resourceProject builds a temp project root with go.mod, a copy of
// golden/backend/example and, when withBruno, golden/docs/bruno/example.
func resourceProject(t *testing.T, withBruno bool) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module golden-app\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}
	golden := filepath.Join("..", "..", "golden")
	copyTree(t, filepath.Join(golden, "backend", "example"), filepath.Join(root, "backend", "example"))
	if withBruno {
		copyTree(t, filepath.Join(golden, "docs", "bruno", "example"), filepath.Join(root, "docs", "bruno", "example"))
	}
	return root
}

// snapshot maps every file under dir (root-relative slash path) to its bytes.
func snapshot(t *testing.T, root string, dirs ...string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	for _, dir := range dirs {
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(dir)), func(p string, d os.DirEntry, err error) error {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			if err != nil || d.IsDir() {
				return err
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, p)
			out[filepath.ToSlash(rel)] = data
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", dir, err)
		}
	}
	return out
}

func TestResource_Golden(t *testing.T) {
	tests := []struct {
		name           string
		spec           ResourceSpec
		withBruno      bool
		wantMigration  string
		wantRenumbered []string
		wantBruno      []string
	}{
		{
			name:          "product-all",
			spec:          ResourceSpec{Domain: "example", Name: "product", Fields: []string{"title:string", "price:int"}},
			withBruno:     true,
			wantMigration: "backend/example/migrations/0002_create_products.go",
			wantRenumbered: []string{
				"docs/bruno/example/Create Stub (Empty Name).bru",
				"docs/bruno/example/Create Stub.bru",
				"docs/bruno/example/Get Stub (Not Found).bru",
				"docs/bruno/example/Get Stub.bru",
				"docs/bruno/example/Login (Example Setup).bru",
				"docs/bruno/example/Logout (Example Teardown).bru",
			},
			wantBruno: []string{
				"Create Product (No Session)", "Create Product", "Create Product (Missing Field)",
				"Get Product (No Session)", "Get Product", "Get Product (Not Found)",
				"List Products (No Session)", "List Products",
				"Update Product (No Session)", "Update Product", "Update Product (Not Found)",
				"Delete Product (No Session)", "Delete Product (Not Found)", "Delete Product",
			},
		},
		{
			name: "widget-public",
			spec: ResourceSpec{
				Domain: "example", Name: "widget", Auth: "public", Ops: []string{"create", "get"},
				Fields: []string{"name:string", "count:int", "total:int64", "ratio:float", "active:bool", "due:time"},
			},
			withBruno:      true,
			wantMigration:  "backend/example/migrations/0002_create_widgets.go",
			wantRenumbered: []string{"docs/bruno/example/Logout (Example Teardown).bru"},
			wantBruno:      []string{"Create Widget", "Create Widget (Missing Field)", "Get Widget", "Get Widget (Not Found)"},
		},
		{
			name:           "category-list",
			spec:           ResourceSpec{Domain: "example", Name: "category", Plural: "categories", Auth: "mutations", Ops: []string{"list"}, Fields: []string{"name:string"}},
			withBruno:      true,
			wantMigration:  "backend/example/migrations/0002_create_categories.go",
			wantRenumbered: []string{"docs/bruno/example/Logout (Example Teardown).bru"},
			wantBruno:      []string{"List Categories"},
		},
		{
			name:          "tag-nofields",
			spec:          ResourceSpec{Domain: "example", Name: "tag"},
			withBruno:     false,
			wantMigration: "backend/example/migrations/0002_create_tags.go",
			wantBruno: []string{
				"Register (Example Setup)", "Confirm Email (Example Setup)", "Login (Example Setup)",
				"Create Tag (No Session)", "Create Tag",
				"Get Tag (No Session)", "Get Tag", "Get Tag (Not Found)",
				"List Tags (No Session)", "List Tags",
				"Update Tag (No Session)", "Update Tag", "Update Tag (Not Found)",
				"Delete Tag (No Session)", "Delete Tag (Not Found)", "Delete Tag",
				"Logout (Example Teardown)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := resourceProject(t, tt.withBruno)
			before := snapshot(t, root, "backend/example", "docs/bruno/example")
			got, err := Resource(root, tt.spec)
			if err != nil {
				t.Fatalf("Resource: unexpected error: %v", err)
			}

			name := tt.spec.Name
			wantCreated := []string{
				"backend/example/domain/" + name + ".go",
				"backend/example/internal/application/" + name + "_service.go",
				"backend/example/internal/application/" + name + "_service_test.go",
				"backend/example/internal/infrastructure/memory/" + name + "_repository.go",
				"backend/example/internal/infrastructure/memory/" + name + "_repository_test.go",
				"backend/example/internal/infrastructure/postgres/" + name + "_repository.go",
				"backend/example/internal/infrastructure/postgres/" + name + "_repository_test.go",
				"backend/example/internal/presentation/" + name + ".go",
				"backend/example/internal/presentation/" + name + "_test.go",
				tt.wantMigration,
			}
			for _, b := range tt.wantBruno {
				wantCreated = append(wantCreated, "docs/bruno/example/"+b+".bru")
			}
			if strings.Join(got.Created, "\n") != strings.Join(wantCreated, "\n") {
				t.Errorf("Created:\n--- want\n%s\n--- got\n%s", strings.Join(wantCreated, "\n"), strings.Join(got.Created, "\n"))
			}
			if got.Facade != "backend/example/example.go" || got.Migration != tt.wantMigration {
				t.Errorf("Facade = %q, Migration = %q", got.Facade, got.Migration)
			}
			if strings.Join(got.Renumbered, "\n") != strings.Join(tt.wantRenumbered, "\n") {
				t.Errorf("Renumbered = %v, want %v", got.Renumbered, tt.wantRenumbered)
			}

			// Every golden file matches its counterpart; every other file
			// under the two directories is an untouched original.
			caseDir := filepath.Join("testdata", "resource", tt.name)
			expected := snapshot(t, caseDir, "backend/example", "docs/bruno/example")
			after := snapshot(t, root, "backend/example", "docs/bruno/example")
			for rel, want := range expected {
				gotData, ok := after[rel]
				if !ok {
					t.Errorf("%s: not written", rel)
					continue
				}
				if !bytes.Equal(gotData, want) {
					t.Errorf("%s: content differs\n--- want\n%s\n--- got\n%s", rel, want, gotData)
				}
			}
			for rel, gotData := range after {
				if _, ok := expected[rel]; ok {
					continue
				}
				orig, ok := before[rel]
				if !ok {
					t.Errorf("%s: unexpected file written", rel)
				} else if !bytes.Equal(gotData, orig) {
					t.Errorf("%s: original was modified", rel)
				}
			}
		})
	}
}

func TestResource_Errors(t *testing.T) {
	const handAdd = "\nadd to Register by hand:\n" +
		"\tproductRepo := postgres.NewProductRepository(db)\n" +
		"\tproductSvc := application.NewProductService(productRepo)\n" +
		"\tpresentation.RegisterProduct(api, productSvc, logger)"
	product := ResourceSpec{Domain: "example", Name: "product", Fields: []string{"title:string"}}
	tests := []struct {
		name    string
		spec    ResourceSpec
		setup   func(t *testing.T, root string)
		wantErr string
	}{
		{"unknown domain", ResourceSpec{Domain: "orders", Name: "order"}, nil, "no such domain backend/orders"},
		{"internal is not a domain", ResourceSpec{Domain: "internal", Name: "order"}, nil, "no such domain backend/internal"},
		{"bad name", ResourceSpec{Domain: "example", Name: "Product"}, nil, `invalid resource name "Product": use snake_case`},
		{"bad field", ResourceSpec{Domain: "example", Name: "product", Fields: []string{"title"}}, nil, `invalid field "title": use name:type`},
		{"implicit field", ResourceSpec{Domain: "example", Name: "product", Fields: []string{"id:string"}}, nil, `field "id" is implicit`},
		{"unknown type", ResourceSpec{Domain: "example", Name: "product", Fields: []string{"x:uuid"}}, nil, `unknown type "uuid" for field "x": use string, int, int64, float, bool or time`},
		{"unknown op", ResourceSpec{Domain: "example", Name: "product", Ops: []string{"patch"}}, nil, `unknown operation "patch": use create, get, list, update or delete`},
		{"unknown auth", ResourceSpec{Domain: "example", Name: "product", Auth: "owner"}, nil, `unknown auth policy "owner": use all, mutations or public`},
		{
			"facade without Register", product,
			func(t *testing.T, root string) {
				in := readFacadeCase(t, "no-register", "in.go")
				if err := os.WriteFile(filepath.Join(root, "backend", "example", "example.go"), in, 0o644); err != nil {
					t.Fatalf("rewriting facade: %v", err)
				}
			},
			"cannot wire product into backend/example/example.go: no func Register" + handAdd,
		},
		{
			"facade missing", product,
			func(t *testing.T, root string) {
				if err := os.Remove(filepath.Join(root, "backend", "example", "example.go")); err != nil {
					t.Fatalf("removing facade: %v", err)
				}
			},
			"cannot wire product into backend/example/example.go: no such file" + handAdd,
		},
		{
			"first target exists", product,
			func(t *testing.T, root string) {
				if err := os.WriteFile(filepath.Join(root, "backend", "example", "domain", "product.go"), []byte("package domain\n"), 0o644); err != nil {
					t.Fatalf("pre-creating: %v", err)
				}
			},
			"backend/example/domain/product.go already exists",
		},
		{
			"last go target exists", product,
			func(t *testing.T, root string) {
				if err := os.WriteFile(filepath.Join(root, "backend", "example", "internal", "presentation", "product_test.go"), []byte("package presentation\n"), 0o644); err != nil {
					t.Fatalf("pre-creating: %v", err)
				}
			},
			"backend/example/internal/presentation/product_test.go already exists",
		},
		{
			"bruno target exists", product,
			func(t *testing.T, root string) {
				if err := os.WriteFile(filepath.Join(root, "docs", "bruno", "example", "Create Product.bru"), []byte("meta {\n  seq: 99\n}\n"), 0o644); err != nil {
					t.Fatalf("pre-creating: %v", err)
				}
			},
			"docs/bruno/example/Create Product.bru already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := resourceProject(t, true)
			if tt.setup != nil {
				tt.setup(t, root)
			}
			before := snapshot(t, root, "backend", "docs")
			_, err := Resource(root, tt.spec)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Resource: err = %q, want %q", err, tt.wantErr)
			}

			after := snapshot(t, root, "backend", "docs")
			if len(after) != len(before) {
				t.Fatalf("Resource: file count changed from %d to %d", len(before), len(after))
			}
			for rel, data := range after {
				if !bytes.Equal(data, before[rel]) {
					t.Errorf("%s: changed on a failed generation", rel)
				}
			}
		})
	}
}
