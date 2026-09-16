package generate

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRoutesFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", full, err)
	}
}

func TestRoutes(t *testing.T) {
	root := pageProject(t)
	// Two files in app/ reference two different calls: one entry each,
	// deduped to a single "app" directory.
	writeRoutesFixture(t, root, "frontend/app/page.tsx", `import { api } from "@/lib/api";
api.system.healthz();
`)
	writeRoutesFixture(t, root, "frontend/app/actions.ts", `import { api } from "@/lib/api";
api.users.getCurrentUser();
`)
	// A nested route references one call.
	writeRoutesFixture(t, root, "frontend/app/login/actions.ts", `import { api } from "@/lib/api";
api.users.loginUser();
`)
	// node_modules must be ignored even though it contains a matching call.
	writeRoutesFixture(t, root, "frontend/app/node_modules/x/index.ts", `api.users.loginUser();`)

	routes, err := Routes(root)
	if err != nil {
		t.Fatalf("Routes: unexpected error: %v", err)
	}

	byID := make(map[string]Route)
	for _, r := range routes {
		byID[r.OperationID] = r
	}

	// Row order and call names come from Document.Operations(): path
	// ascending, then the fixed method order.
	if routes[0].OperationID != "healthz" || routes[0].Method != "GET" || routes[0].Path != "/healthz" {
		t.Errorf("routes[0] = %+v, want GET /healthz (healthz)", routes[0])
	}

	healthz, ok := byID["healthz"]
	if !ok {
		t.Fatal("healthz route not found")
	}
	if healthz.CallName != "api.system.healthz" {
		t.Errorf("healthz.CallName = %q, want api.system.healthz", healthz.CallName)
	}
	if got := healthz.UsedBy; len(got) != 1 || got[0] != "app" {
		t.Errorf("healthz.UsedBy = %v, want [app]", got)
	}

	getCurrentUser, ok := byID["get-current-user"]
	if !ok {
		t.Fatal("get-current-user route not found")
	}
	if got := getCurrentUser.UsedBy; len(got) != 1 || got[0] != "app" {
		t.Errorf("get-current-user.UsedBy = %v, want [app]", got)
	}

	loginUser, ok := byID["login-user"]
	if !ok {
		t.Fatal("login-user route not found")
	}
	if got := loginUser.UsedBy; len(got) != 1 || got[0] != "app/login" {
		t.Errorf("login-user.UsedBy = %v, want [app/login]", got)
	}

	// An operation nothing references gets an empty UsedBy.
	getStub, ok := byID["get-stub"]
	if !ok {
		t.Fatal("get-stub route not found")
	}
	if len(getStub.UsedBy) != 0 {
		t.Errorf("get-stub.UsedBy = %v, want empty", getStub.UsedBy)
	}
}

func TestRoutes_NoFrontendApp(t *testing.T) {
	root := pageProject(t)
	if err := os.RemoveAll(filepath.Join(root, "frontend", "app")); err != nil {
		t.Fatalf("removing frontend/app: %v", err)
	}

	routes, err := Routes(root)
	if err != nil {
		t.Fatalf("Routes: unexpected error: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("Routes: got no rows, want the full operation list")
	}
	for _, r := range routes {
		if len(r.UsedBy) != 0 {
			t.Errorf("%s.UsedBy = %v, want empty (no frontend/app)", r.OperationID, r.UsedBy)
		}
	}
}

func TestRoutes_MissingDocument(t *testing.T) {
	root := t.TempDir()

	_, err := Routes(root)
	if err == nil {
		t.Fatal("Routes: expected error, got nil")
	}
	want := "docs/openapi.yaml not found; run `gonext openapi`"
	if err.Error() != want {
		t.Errorf("Routes: err = %q, want %q", err.Error(), want)
	}
}
