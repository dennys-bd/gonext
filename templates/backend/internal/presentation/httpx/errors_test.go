package httpx_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"

	"[PROJECT-NAME]/backend/internal/presentation/httpx"
)

var (
	errNotFound = errors.New("stub not found")
	errInvalid  = errors.New("stub name is required")
)

// secretText stands in for wrapped internals that must never reach a client;
// it carries no quotes, so it survives slog's escaping for a plain substring match.
const secretText = "pq: password authentication failed for user golden"

// problem is the RFC 9457 body huma renders a StatusError into.
type problem struct {
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// failingAPI builds a one-route API whose handler always returns err; logs
// captures whatever the group's logger writes.
func failingAPI(t *testing.T, err error) (humatest.TestAPI, *bytes.Buffer) {
	t.Helper()

	logs := &bytes.Buffer{}
	_, api := humatest.New(t)
	g := httpx.NewGroup(api, "/stubs", "Example", slog.New(slog.NewTextHandler(logs, nil))).Errors(
		httpx.Map(errInvalid, http.StatusBadRequest),
		httpx.Map(errNotFound, http.StatusNotFound),
	)

	httpx.Get(g, "/{id}", "get-stub", func(*httpx.Ctx, *groupInput) (*groupOutput, error) {
		return nil, err
	})

	return api, logs
}

func decodeProblem(t *testing.T, body string) problem {
	t.Helper()

	var p problem
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("body is not a problem document: %v (%s)", err, body)
	}
	return p
}

func TestGroupErrors_MappedSentinelKeepsItsMessage(t *testing.T) {
	api, _ := failingAPI(t, errNotFound)

	resp := api.Get("/stubs/abc")

	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", resp.Code, resp.Body.String())
	}
	if got := decodeProblem(t, resp.Body.String()).Detail; got != errNotFound.Error() {
		t.Errorf("detail = %q, want %q", got, errNotFound.Error())
	}
}

// Mappings match with errors.Is, so a sentinel wrapped with %w for context still maps.
func TestGroupErrors_WrappedSentinelStillMaps(t *testing.T) {
	api, _ := failingAPI(t, fmt.Errorf("loading stub abc: %w", errInvalid))

	resp := api.Get("/stubs/abc")

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", resp.Code, resp.Body.String())
	}
	// Wrapped context is for the log only and must not travel to the client.
	problem := decodeProblem(t, resp.Body.String())
	if problem.Detail != errInvalid.Error() {
		t.Errorf("detail = %q, want exactly %q", problem.Detail, errInvalid.Error())
	}
	if strings.Contains(resp.Body.String(), "loading stub abc") {
		t.Errorf("wrapped context leaked into the response: %s", resp.Body.String())
	}
}

func TestGroupErrors_UnmappedErrorIsLoggedAndFlattened(t *testing.T) {
	api, logs := failingAPI(t, fmt.Errorf("querying stubs: %w", errors.New(secretText)))

	resp := api.Get("/stubs/abc")
	body := resp.Body.String()

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", resp.Code, body)
	}

	p := decodeProblem(t, body)
	if p.Detail != "internal server error" {
		t.Errorf("detail = %q, want %q", p.Detail, "internal server error")
	}
	for _, e := range p.Errors {
		if strings.Contains(e.Message, secretText) || strings.Contains(e.Message, "querying stubs") {
			t.Errorf("errors[].message leaked the original error: %q", e.Message)
		}
	}
	if strings.Contains(body, secretText) || strings.Contains(body, "querying stubs") {
		t.Errorf("response body leaked the original error: %s", body)
	}

	if logged := logs.String(); !strings.Contains(logged, secretText) {
		t.Errorf("the original error was not logged; log was: %s", logged)
	}
}

// A StatusError reached through a wrapper is unwrapped, so the wrapping text
// itself (which huma would otherwise render into the body) never ships.
func TestGroupErrors_WrappedStatusErrorDropsTheWrapper(t *testing.T) {
	api, _ := failingAPI(t, fmt.Errorf("querying stubs for tenant acme: %w", huma.Error404NotFound("stub not found")))

	resp := api.Get("/stubs/abc")

	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", resp.Code, resp.Body.String())
	}
	if got := decodeProblem(t, resp.Body.String()).Detail; got != "stub not found" {
		t.Errorf("detail = %q, want exactly %q", got, "stub not found")
	}
	if strings.Contains(resp.Body.String(), "tenant acme") {
		t.Errorf("wrapper context leaked into the response: %s", resp.Body.String())
	}
}

func TestGroupErrors_StatusErrorPassesThrough(t *testing.T) {
	api, _ := failingAPI(t, huma.Error409Conflict("stub already exists"))

	resp := api.Get("/stubs/abc")

	if resp.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409: %s", resp.Code, resp.Body.String())
	}
	if got := decodeProblem(t, resp.Body.String()).Detail; got != "stub already exists" {
		t.Errorf("detail = %q, want %q", got, "stub already exists")
	}
}

// matchesBoth matches both declared sentinels, to test that declaration order decides.
type matchesBoth struct{}

func (matchesBoth) Error() string { return "matches both sentinels" }

func (matchesBoth) Is(target error) bool {
	return target == errInvalid || target == errNotFound
}

// Repeated: mappings are a slice, not a map, so the order must be stable across runs.
func TestGroupErrors_FirstDeclarationWins(t *testing.T) {
	const runs = 50

	for i := range runs {
		api, _ := failingAPI(t, matchesBoth{})

		resp := api.Get("/stubs/abc")

		// errInvalid is declared first, so 400 beats 404 every time.
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("run %d: status = %d, want 400 from the first declared mapping: %s",
				i, resp.Code, resp.Body.String())
		}
	}
}
