package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/dennys-bd/gonext/internal/generate"
)

func TestRunRoutes_RejectsArgs(t *testing.T) {
	if got := runRoutes([]string{"extra"}); got != 1 {
		t.Errorf("runRoutes([]string{\"extra\"}) = %d, want 1", got)
	}
}

var twoSpaceRE = regexp.MustCompile(`  +`)

func TestPrintRoutes(t *testing.T) {
	routes := []generate.Route{
		{OperationID: "healthz", Method: "GET", Path: "/healthz", CallName: "api.system.healthz"},
		{OperationID: "get-stub", Method: "GET", Path: "/stubs/{id}", CallName: "api.example.getStub", UsedBy: []string{"app", "app/stubs/[id]"}},
	}

	var buf bytes.Buffer
	printRoutes(&buf, routes)
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")

	if len(lines) != 3 {
		t.Fatalf("printRoutes: got %d lines, want 3 (header + 2 rows):\n%s", len(lines), buf.String())
	}
	// tabwriter pads each column to its widest cell, so compare the
	// header with the runs of two-or-more spaces collapsed.
	header := twoSpaceRE.ReplaceAllString(lines[0], "\t")
	if want := "operationId\tmethod\tpath\tclient call\tused by"; header != want {
		t.Errorf("printRoutes: header = %q, want %q", lines[0], want)
	}

	healthzRow := strings.Fields(lines[1])
	if got, want := healthzRow[len(healthzRow)-1], "—"; got != want {
		t.Errorf("printRoutes: healthz row's used-by column = %q, want %q (row: %q)", got, want, lines[1])
	}

	if !strings.Contains(lines[2], "app, app/stubs/[id]") {
		t.Errorf("printRoutes: get-stub row = %q, want it to contain %q", lines[2], "app, app/stubs/[id]")
	}
}
