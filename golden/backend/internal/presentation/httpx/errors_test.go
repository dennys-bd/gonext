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

	"golden-app/backend/internal/presentation/httpx"
)

var (
	errNotFound = errors.New("stub not found")
	errInvalid  = errors.New("stub name is required")
)

// secretText stands in for the wrapped internals — Postgres driver
// output, connection strings — that must never reach a client. It
// carries no quotes of its own, so it survives slog's escaping
// unchanged and the log assertion below stays a plain substring match.
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

// failingAPI builds a one-route API whose handler always returns err,
// registered through a group declaring the two mappings below. logs
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

// A sentinel the group declared becomes its declared status, and the
// domain's own message survives as the client-facing detail — that
// message is written for the caller, so flattening it would be a
// regression, not extra safety.
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

// Mappings are matched with errors.Is, so a sentinel a service wrapped
// with %w for context still maps. Without this, every service would
// have to return bare sentinels to stay mappable.
func TestGroupErrors_WrappedSentinelStillMaps(t *testing.T) {
	api, _ := failingAPI(t, fmt.Errorf("loading stub abc: %w", errInvalid))

	resp := api.Get("/stubs/abc")

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", resp.Code, resp.Body.String())
	}
	// The sentinel's own message is what the client gets. The context a
	// caller wrapped around it is for the log, and must not travel: the
	// same %w that adds "loading stub abc" here could just as easily add
	// a connection string.
	problem := decodeProblem(t, resp.Body.String())
	if problem.Detail != errInvalid.Error() {
		t.Errorf("detail = %q, want exactly %q", problem.Detail, errInvalid.Error())
	}
	if strings.Contains(resp.Body.String(), "loading stub abc") {
		t.Errorf("wrapped context leaked into the response: %s", resp.Body.String())
	}
}

// The leak barrier: an error the group did not declare is logged
// server-side and returned as a flat 500 carrying none of its text.
// huma renders a non-StatusError by putting err.Error() into the
// response, so without this wrapper wrapped internals ship to the
// client.
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

// A handler can still answer with a status directly for a case not
// worth declaring on the group; the wrapper passes that through
// untouched rather than flattening it to 500.
// A StatusError reached through a wrapper is unwrapped before it is
// returned. This is the leak-relevant half of the passthrough rule: if
// the wrapper itself were returned, huma would render the wrapping
// text — "querying stubs: ..." — into the response body, which is
// exactly what the group exists to prevent.
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

// matchesBoth reports itself as both declared sentinels, forcing the
// ordering question: whichever mapping is declared first must win.
type matchesBoth struct{}

func (matchesBoth) Error() string { return "matches both sentinels" }

func (matchesBoth) Is(target error) bool {
	return target == errInvalid || target == errNotFound
}

// Mappings are walked in declaration order on every request. This is
// why Errors takes an ordered slice and not a map[error]int: map
// iteration is randomised, so the status would vary between requests.
// Repeated to be meaningful against that randomness.
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
