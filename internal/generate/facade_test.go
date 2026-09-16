package generate

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func facadeModel(t *testing.T, domain string) resourceModel {
	t.Helper()
	m, err := buildResourceModel("golden-app", ResourceSpec{Domain: domain, Name: "product"})
	if err != nil {
		t.Fatalf("buildResourceModel: %v", err)
	}
	return m
}

func readFacadeCase(t *testing.T, name, file string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "resource", "facade", name, file))
	if err != nil {
		t.Fatalf("reading %s/%s: %v", name, file, err)
	}
	return data
}

func TestWireFacade_Golden(t *testing.T) {
	tests := []struct {
		name   string
		domain string
	}{
		{"example", "example"},
		{"renamed-params", "example"},
		{"extra-params", "users"},
		{"imports-partial", "example"},
		{"single-line-imports", "example"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := facadeModel(t, tt.domain)
			rel := "backend/" + tt.domain + "/" + tt.domain + ".go"
			in := readFacadeCase(t, tt.name, "in.go")
			want := readFacadeCase(t, tt.name, "want.go")

			info, err := inspectFacade(rel, in, m)
			if err != nil {
				t.Fatalf("inspectFacade: unexpected error: %v", err)
			}
			got, err := wireFacade(in, info, m)
			if err != nil {
				t.Fatalf("wireFacade: unexpected error: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("content differs\n--- want\n%s\n--- got\n%s", want, got)
			}
		})
	}
}

func TestInspectFacade_Errors(t *testing.T) {
	const prefix = "cannot wire product into backend/example/example.go: "
	handAdd := func(api, db, logger string) string {
		return "\nadd to Register by hand:\n" +
			"\tproductRepo := postgres.NewProductRepository(" + db + ")\n" +
			"\tproductSvc := application.NewProductService(productRepo)\n" +
			"\tpresentation.RegisterProduct(" + api + ", productSvc, " + logger + ")"
	}
	tests := []struct {
		name    string
		wantErr string
	}{
		{"no-register", prefix + "no func Register" + handAdd("api", "db", "logger")},
		{"no-api-param", prefix + "Register has no huma.API parameter" + handAdd("api", "db", "logger")},
		{"no-db-param", prefix + "Register has no *bun.DB parameter" + handAdd("a", "db", "log")},
		{"no-logger-param", prefix + "Register has no *slog.Logger parameter" + handAdd("api", "db", "logger")},
		{"no-return-nil", prefix + "Register does not end with return nil" + handAdd("api", "db", "logger")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := facadeModel(t, "example")
			in := readFacadeCase(t, tt.name, "in.go")

			_, err := inspectFacade("backend/example/example.go", in, m)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("inspectFacade: err = %q, want %q", err, tt.wantErr)
			}
		})
	}
}
