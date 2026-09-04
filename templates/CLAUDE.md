# [PROJECT-NAME]

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

Register every endpoint with `httpx.Register`, never `huma.Register` — the wrapper is what gives the handler a `*httpx.Ctx`, and a `forbidigo` lint rule enforces it.

An operation declares what it needs in its `Security` field; the auth middleware enforces that before the handler runs, and injects the identity:

```go
httpx.Register(api, huma.Operation{
    OperationID: "delete-post",
    Method:      http.MethodDelete,
    Path:        "/posts/{id}",
    Security:    auth.RequirePermission("posts:delete"),
}, func(ctx *httpx.Ctx, in *deleteInput) (*struct{}, error) {
    return nil, svc.Delete(ctx, ctx.Identity().UserID, in.ID)
})
```

| Declaration | Behaviour |
|---|---|
| *(no `Security` field)* | Public. The credential is never read, so there is no session lookup. |
| `auth.Required()` | Any valid session. 401 without one. |
| `auth.RequireRole("admin")` | Exact role match. 401 without a session, 403 with the wrong role. |
| `auth.RequirePermission("posts:delete")` | Permission held via the role. 401 without a session, 403 without the permission. |
| `auth.Optional()` | Resolves an identity when a credential is present, never rejects. Read it with `ctx.IdentityOK()`. |

`ctx.Identity()` is guaranteed on the three requiring declarations and **panics** on `auth.Optional()` or an undeclared operation — use `ctx.IdentityOK()` there.

`auth` here is `github.com/dennys-bd/gonext/auth`, the one imported (rather than scaffolded) part of this project. Changing identity provider means providing a different `auth.Resolver` in `backend/wire.go`; nothing else moves.

Endpoints requiring a session need their `.bru` file to send `Cookie: {{...SessionCookie}}`, plus a sibling `(No Session)` request asserting 401 — see `docs/bruno/example/`.
