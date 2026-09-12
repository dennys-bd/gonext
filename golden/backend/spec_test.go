package main

import (
	"bytes"
	"fmt"
	"testing"
)

// wantOperationIDs mirrors every httpx.Register operation ID declared
// across the backend's tracks — see the plan's "What has moved"
// section. If a new endpoint is registered without a matching update
// here, this test still passes; if an existing one is renamed or
// dropped without the spec being regenerated, it fails loudly.
var wantOperationIDs = []string{
	"healthz",
	"readyz",
	"create-stub",
	"get-stub",
	"register-user",
	"login-user",
	"logout-user",
	"get-current-user",
	"confirm-user-email",
	"request-password-reset",
	"confirm-password-reset",
}

func TestInitializeSpec_ContainsEveryOperationID(t *testing.T) {
	spec, err := InitializeSpec()
	if err != nil {
		t.Fatalf("InitializeSpec: unexpected error: %v", err)
	}

	doc, err := spec.API.OpenAPI().YAML()
	if err != nil {
		t.Fatalf("YAML: unexpected error: %v", err)
	}

	for _, id := range wantOperationIDs {
		marker := []byte(fmt.Sprintf("operationId: %s", id))
		if !bytes.Contains(doc, marker) {
			t.Errorf("document missing operation %q", id)
		}
	}
}

func TestInitializeSpec_DeterministicAcrossCalls(t *testing.T) {
	first, err := InitializeSpec()
	if err != nil {
		t.Fatalf("InitializeSpec (first): unexpected error: %v", err)
	}
	firstYAML, err := first.API.OpenAPI().YAML()
	if err != nil {
		t.Fatalf("YAML (first): unexpected error: %v", err)
	}

	second, err := InitializeSpec()
	if err != nil {
		t.Fatalf("InitializeSpec (second): unexpected error: %v", err)
	}
	secondYAML, err := second.API.OpenAPI().YAML()
	if err != nil {
		t.Fatalf("YAML (second): unexpected error: %v", err)
	}

	if !bytes.Equal(firstYAML, secondYAML) {
		t.Fatal("two InitializeSpec calls produced different documents")
	}
}

func TestInitializeSpec_UnaffectedByEnvironment(t *testing.T) {
	baseline, err := InitializeSpec()
	if err != nil {
		t.Fatalf("InitializeSpec (baseline): unexpected error: %v", err)
	}
	baselineYAML, err := baseline.API.OpenAPI().YAML()
	if err != nil {
		t.Fatalf("YAML (baseline): unexpected error: %v", err)
	}

	t.Setenv("DATABASE_URL", "postgres://nope")
	t.Setenv("ENV", "dev")
	t.Setenv("PORT", "1")

	withEnv, err := InitializeSpec()
	if err != nil {
		t.Fatalf("InitializeSpec (with hostile env): unexpected error: %v", err)
	}
	withEnvYAML, err := withEnv.API.OpenAPI().YAML()
	if err != nil {
		t.Fatalf("YAML (with hostile env): unexpected error: %v", err)
	}

	if !bytes.Equal(baselineYAML, withEnvYAML) {
		t.Fatal("InitializeSpec output changed under hostile environment variables")
	}
}
