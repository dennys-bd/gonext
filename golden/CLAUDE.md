# golden-app

Monorepo root. See `README.md` for the app layout and `docs/superpowers/specs/` for architecture decisions.

## Bruno collection (docs/bruno/)

Every HTTP endpoint exposed by `backend/` must have a matching Bruno request file under `docs/bruno/`, kept in sync whenever an endpoint is added, changed, or removed:

- Cross-cutting endpoints (not owned by any domain, e.g. `/healthz`) live directly under `docs/bruno/` (see `Healthz.bru`).
- Domain endpoints live under `docs/bruno/<Domain>/` matching the domain's folder name in `backend/` (see `example/` for the `example` domain's `/stubs` endpoints).
- Use `{{baseUrl}}` (defined in `docs/bruno/environments/Local.bru`) instead of hardcoding a host.
- Add an `assert { res.status: eq <code> }` block for the expected status code.
- Add a `tests { test("...", function () { expect(...) }) }` block for meaningful response-body checks (e.g. a created resource's id/fields, or an error's `detail` message) — this is what makes the collection double as the project's smoke test.
- Cover both the happy path and the domain's documented error cases (e.g. not-found, validation failure) as separate request files, following the naming pattern `<Action> (<Case>).bru`.
- When one request's response feeds another (e.g. a created resource's id), chain it with `script:post-response { bru.setVar("...", res.body...) }` in the producer and `{{varName}}` / `bru.getVar("...")` in the consumer, rather than hardcoding example values.

When implementing a new endpoint or changing an existing one's request/response shape or status codes, update the corresponding `.bru` file(s) in the same change — don't leave this as a follow-up.

## Smoke testing

`make smoke` is the project's smoke test: it builds and runs the backend server, then runs the entire `docs/bruno/` collection against it via `mise exec -- bru run . --env Local -r` (run from inside `docs/bruno/` — the Bruno CLI only runs from a collection's own root), and stops the server afterward. A passing `make smoke` means every request in the collection returned its expected status and passed its `tests` block.

The `bru` CLI (`npm:@usebruno/cli`) is pinned in `.mise.toml` and provisioned by `mise install`. It's invoked via `mise exec --` rather than assumed to be on `PATH`, because Makefile recipes run in a non-interactive shell where mise's dynamic PATH hook (which only updates on an interactive prompt) hasn't necessarily fired yet.

## Guarding an endpoint

Routes are registered on an `httpx.Group`, which owns the path prefix, the OpenAPI tag, and the error policy that every operation on the track shares. Never call `huma.Register` — a `forbidigo` lint rule forbids it, because the group and `httpx.Register` are what give a handler a `*httpx.Ctx`.

```go
func RegisterStub(api huma.API, svc *application.StubService, logger *slog.Logger) {
    h := &handlers{svc: svc}
    g := httpx.NewGroup(api, "/stubs", "Example", logger).Errors(
        httpx.Map(domain.ErrStubNameRequired, http.StatusBadRequest),
        httpx.Map(domain.ErrStubNotFound, http.StatusNotFound),
    )

    httpx.Post(g, "", "create-stub", h.createStub,
        httpx.Summary("Create a stub"),
        httpx.Status(http.StatusCreated),
        httpx.Secured(auth.Required()))

    httpx.Get(g, "/{id}", "get-stub", h.getStub,
        httpx.Summary("Get a stub by id"))
}
```

The verbs are `httpx.Get`, `Post`, `Put`, `Patch` and `Delete`. They take the group as their first argument rather than being methods on it because Go does not permit methods to have type parameters, and each route is generic over its own input and output types.

The operation ID is explicit and positional, never derived: it becomes the generated TypeScript client's method name, so `api.createStub()` is worth writing by hand.

**A collection-root route passes the empty string, not `"/"`.** The group composes `prefix + path` verbatim, so `"/"` yields `/stubs/` — a different route with different behaviour.

Options are `httpx.Summary`, `Description`, `Status` (the default success status), `Secured`, and `Deprecated`.

`NewGroup` requires a non-nil logger and panics at construction without one, so a wiring mistake fails the process at boot rather than surfacing as a panic on the 500 path mid-incident.

### Errors

A handler returns its error bare — `return nil, err` — and the group decides what the client sees, in this order:

1. A sentinel declared in `Errors(...)`, matched with `errors.Is` in declaration order, becomes its declared status, and the client is sent **the sentinel's own message**. Declaration order is why the mappings are an ordered list and not a map: map iteration is randomised, so an error matching two sentinels would otherwise get a status that varied between requests.

    Any context wrapped around the sentinel is dropped from the response and appears only in the log, so `fmt.Errorf("loading order %s: %w", id, domain.ErrOrderNotFound)` is safe to write — the same `%w` that adds an order id could just as easily add a connection string, and nothing here can tell the two apart. To send the client a message built at request time, declare a separate sentinel for that case, or return a `huma.ErrorNNN` directly, which step 2 passes through untouched.
2. Otherwise an error already carrying a status — `huma.Error409Conflict(...)` returned directly — passes through, for a case not worth declaring on the group.
3. Otherwise the error is logged with the request context and replaced by a flat 500 carrying none of its detail.

Step 3 is why returning a bare error is safe, and it is not optional: Huma renders an error that carries no status by putting `err.Error()` into the response body, so without it a wrapped internal — database driver text included — would reach the client. Add a `Map` declaration for every domain sentinel that should be something other than a 500.

`httpx.Register` remains available as the escape hatch for anything the group does not model; it takes a full `huma.Operation` and applies no error policy, so such a handler must return its own `huma.ErrorNNN`.

### Security

An operation declares what it needs through `httpx.Secured`; the auth middleware enforces that before the handler runs, and injects the identity:

```go
httpx.Delete(g, "/{id}", "delete-post", h.deletePost,
    httpx.Secured(auth.RequirePermission("posts:delete")))
```

| Declaration | Behaviour |
|---|---|
| *(no `httpx.Secured` option)* | Public. The credential is never read, so there is no session lookup. |
| `auth.Required()` | Any valid session. 401 without one. |
| `auth.RequireRole("admin")` | Exact role match. 401 without a session, 403 with the wrong role. |
| `auth.RequirePermission("posts:delete")` | Permission held via the role. 401 without a session, 403 without the permission. |
| `auth.Optional()` | Resolves an identity when a credential is present, never rejects. Read it with `ctx.IdentityOK()`. |

`ctx.Identity()` is guaranteed on the three requiring declarations and **panics** on `auth.Optional()` or an undeclared operation — use `ctx.IdentityOK()` there.

`auth` here is `github.com/dennys-bd/gonext/auth`, the one imported (rather than scaffolded) part of this project. Changing identity provider means providing a different `auth.Resolver` in `backend/wire.go`; nothing else moves.

Endpoints requiring a session need their `.bru` file to send `Cookie: {{...SessionCookie}}`, plus a sibling `(No Session)` request asserting 401 — see `docs/bruno/example/`.
