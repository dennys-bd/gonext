package generate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// copyTree copies every regular file under src into dst, creating
// directories as needed.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copying %s: %v", src, err)
	}
}

func brunoExampleCopy(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "example")
	copyTree(t, filepath.Join("..", "..", "golden", "docs", "bruno", "example"), dir)
	return dir
}

func TestBrunoFolder(t *testing.T) {
	t.Run("golden example", func(t *testing.T) {
		st, err := brunoFolder(brunoExampleCopy(t), "Example")
		if err != nil {
			t.Fatalf("brunoFolder: unexpected error: %v", err)
		}
		if st.maxSeq != 9 || st.loginSeq != 4 || st.teardown != "Logout (Example Teardown).bru" {
			t.Fatalf("brunoFolder: state = %+v", st)
		}
		if len(st.seqs) != 9 || st.seqs["Create Stub.bru"] != 5 {
			t.Fatalf("brunoFolder: seqs = %v", st.seqs)
		}
	})
	t.Run("empty folder", func(t *testing.T) {
		st, err := brunoFolder(t.TempDir(), "Example")
		if err != nil {
			t.Fatalf("brunoFolder: unexpected error: %v", err)
		}
		if st.maxSeq != 0 || st.loginSeq != 0 || st.teardown != "" || len(st.seqs) != 0 {
			t.Fatalf("brunoFolder: state = %+v", st)
		}
	})
	t.Run("missing folder", func(t *testing.T) {
		st, err := brunoFolder(filepath.Join(t.TempDir(), "nope"), "Example")
		if err != nil {
			t.Fatalf("brunoFolder: unexpected error: %v", err)
		}
		if st.maxSeq != 0 || st.loginSeq != 0 || st.teardown != "" || len(st.seqs) != 0 {
			t.Fatalf("brunoFolder: state = %+v", st)
		}
	})
}

func TestRenumberSeq(t *testing.T) {
	dir := brunoExampleCopy(t)
	p := filepath.Join(dir, "Logout (Example Teardown).bru")
	before, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}

	if err := renumberSeq(p, 24); err != nil {
		t.Fatalf("renumberSeq: unexpected error: %v", err)
	}

	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	want := bytes.Replace(before, []byte("  seq: 9\n"), []byte("  seq: 24\n"), 1)
	if !bytes.Equal(after, want) {
		t.Fatalf("renumberSeq: content differs\n--- want\n%s\n--- got\n%s", want, after)
	}
}

func brunoModel(t *testing.T, spec ResourceSpec) resourceModel {
	t.Helper()
	spec.Domain = "example"
	m, err := buildResourceModel("golden-app", spec)
	if err != nil {
		t.Fatalf("buildResourceModel: %v", err)
	}
	return m
}

func formatPlan(p brunoPlan) (targets, renumber string) {
	lines := make([]string, 0, len(p.targets))
	for _, tg := range p.targets {
		lines = append(lines, fmt.Sprintf("%s=%d", tg.name, tg.seq))
	}
	targets = strings.Join(lines, "\n")
	lines = lines[:0]
	for name, seq := range p.renumber {
		lines = append(lines, fmt.Sprintf("%s=%d", name, seq))
	}
	sort.Strings(lines)
	return targets, strings.Join(lines, "\n")
}

func TestPlanBruno(t *testing.T) {
	golden := brunoState{
		maxSeq: 9, loginSeq: 4, teardown: "Logout (Example Teardown).bru",
		seqs: map[string]int{
			"Create Stub (No Session).bru":      1,
			"Register (Example Setup).bru":      2,
			"Confirm Email (Example Setup).bru": 3,
			"Login (Example Setup).bru":         4,
			"Create Stub.bru":                   5,
			"Get Stub.bru":                      6,
			"Get Stub (Not Found).bru":          7,
			"Create Stub (Empty Name).bru":      8,
			"Logout (Example Teardown).bru":     9,
		},
	}
	tests := []struct {
		name         string
		spec         ResourceSpec
		state        brunoState
		wantTargets  string
		wantRenumber string
	}{
		{
			// The jar attaches the session, so every (No Session) case is
			// numbered before Login; Login and everything after shift up.
			name:  "all ops, chain present",
			spec:  ResourceSpec{Name: "product", Fields: []string{"title:string"}},
			state: golden,
			wantTargets: strings.Join([]string{
				"Create Product (No Session)=4",
				"Create Product=15",
				"Create Product (Missing Field)=16",
				"Get Product (No Session)=5",
				"Get Product=17",
				"Get Product (Not Found)=18",
				"List Products (No Session)=6",
				"List Products=19",
				"Update Product (No Session)=7",
				"Update Product=20",
				"Update Product (Not Found)=21",
				"Delete Product (No Session)=8",
				"Delete Product (Not Found)=22",
				"Delete Product=23",
			}, "\n"),
			wantRenumber: strings.Join([]string{
				"Create Stub (Empty Name).bru=13",
				"Create Stub.bru=10",
				"Get Stub (Not Found).bru=12",
				"Get Stub.bru=11",
				"Login (Example Setup).bru=9",
				"Logout (Example Teardown).bru=24",
			}, "\n"),
		},
		{
			name:  "all ops, no chain, no fields",
			spec:  ResourceSpec{Name: "tag"},
			state: brunoState{},
			wantTargets: strings.Join([]string{
				"Register (Example Setup)=6",
				"Confirm Email (Example Setup)=7",
				"Login (Example Setup)=8",
				"Create Tag (No Session)=1",
				"Create Tag=9",
				"Get Tag (No Session)=2",
				"Get Tag=10",
				"Get Tag (Not Found)=11",
				"List Tags (No Session)=3",
				"List Tags=12",
				"Update Tag (No Session)=4",
				"Update Tag=13",
				"Update Tag (Not Found)=14",
				"Delete Tag (No Session)=5",
				"Delete Tag (Not Found)=15",
				"Delete Tag=16",
				"Logout (Example Teardown)=17",
			}, "\n"),
			wantRenumber: "",
		},
		{
			name:  "mutations secures writes only",
			spec:  ResourceSpec{Name: "product", Fields: []string{"title:string"}, Auth: "mutations", Ops: []string{"get", "update"}},
			state: golden,
			wantTargets: strings.Join([]string{
				"Get Product=11",
				"Get Product (Not Found)=12",
				"Update Product (No Session)=4",
				"Update Product=13",
				"Update Product (Not Found)=14",
			}, "\n"),
			wantRenumber: strings.Join([]string{
				"Create Stub (Empty Name).bru=9",
				"Create Stub.bru=6",
				"Get Stub (Not Found).bru=8",
				"Get Stub.bru=7",
				"Login (Example Setup).bru=5",
				"Logout (Example Teardown).bru=15",
			}, "\n"),
		},
		{
			name:  "public needs no chain and no no-session cases",
			spec:  ResourceSpec{Name: "widget", Fields: []string{"name:string"}, Auth: "public", Ops: []string{"create", "get"}},
			state: golden,
			wantTargets: strings.Join([]string{
				"Create Widget=10",
				"Create Widget (Missing Field)=11",
				"Get Widget=12",
				"Get Widget (Not Found)=13",
			}, "\n"),
			wantRenumber: "Logout (Example Teardown).bru=14",
		},
		{
			name:         "public with no folder",
			spec:         ResourceSpec{Name: "widget", Auth: "public", Ops: []string{"list"}},
			state:        brunoState{},
			wantTargets:  "List Widgets=1",
			wantRenumber: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := brunoModel(t, tt.spec)
			targets, renumber := formatPlan(planBruno(m, tt.state))
			if targets != tt.wantTargets {
				t.Errorf("targets:\n--- want\n%s\n--- got\n%s", tt.wantTargets, targets)
			}
			if renumber != tt.wantRenumber {
				t.Errorf("renumber:\n--- want\n%s\n--- got\n%s", tt.wantRenumber, renumber)
			}
		})
	}
}

func TestRenderResource_BrunoGolden(t *testing.T) {
	m := productAllModel(t)
	st, err := brunoFolder(brunoExampleCopy(t), m.DomainPascal)
	if err != nil {
		t.Fatalf("brunoFolder: %v", err)
	}

	files, err := renderBruno(m, planBruno(m, st).targets)
	if err != nil {
		t.Fatalf("renderBruno: unexpected error: %v", err)
	}
	if len(files) != 14 {
		t.Fatalf("renderBruno: got %d files, want 14", len(files))
	}
	for _, f := range files {
		if !strings.HasPrefix(f.rel, "docs/bruno/example/") || !strings.HasSuffix(f.rel, ".bru") {
			t.Errorf("unexpected rel %q", f.rel)
		}
		wantData := readGolden(t, "product-all", f.rel)
		if !bytes.Equal(f.data, wantData) {
			t.Errorf("%s: content differs\n--- want\n%s\n--- got\n%s", f.rel, wantData, f.data)
		}
	}
}
