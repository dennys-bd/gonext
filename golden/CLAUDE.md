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
