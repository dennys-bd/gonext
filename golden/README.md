# golden-app

Monorepo root.

## Apps

- [`backend/`](backend/) — Go modular-monolith backend (Echo + Huma). See
  [`docs/superpowers/specs/2026-08-24-monorepo-backend-scaffold-design.md`](docs/superpowers/specs/2026-08-24-monorepo-backend-scaffold-design.md)
  for the architecture.
- [`frontend/`](frontend/) — Next.js frontend (App Router, TypeScript,
  Mantine). See
  [`docs/superpowers/specs/2026-08-25-frontend-scaffold-design.md`](docs/superpowers/specs/2026-08-25-frontend-scaffold-design.md)
  for the architecture.

## Quick start (backend)

```bash
cd backend
go run .
```

In another terminal:

```bash
curl localhost:8080/healthz
```

## Quick start (frontend)

```bash
cd frontend
pnpm install   # also generates lib/api/ from ../docs/openapi.yaml
pnpm dev
```

The frontend calls the backend only from Next's server (server components and server actions) through the generated client in `frontend/lib/api/`. The backend address comes from `API_URL` (server-only, default `http://localhost:8080`); it is never exposed to the browser, so no CORS is configured.

## API contract

`docs/openapi.yaml` is produced by `gonext openapi` from the backend's route registrations, with no database or environment needed:

```bash
make openapi         # regenerate docs/openapi.yaml and frontend/lib/api/
make openapi-check   # fail if docs/openapi.yaml is stale (CI runs this)
```

If `frontend/lib/api/` is missing (it is gitignored), `pnpm --dir frontend codegen` regenerates it.

## Development Prerequisites

Two things on the host: [**`mise`**](https://mise.jdx.dev/) and Docker.

`mise` owns the toolchains — language runtimes and package managers (`go`,
`node`, `pnpm`), the `gonext` CLI, and standalone dev binaries
(`golangci-lint`, `gitleaks`, `lefthook`, `bru`) — pinned in `.mise.toml`.
Dependencies stay with their native tool: `go.mod` (including `go tool`
entries such as `wire` and `govulncheck`) and `pnpm`. Add a new standalone
binary to `.mise.toml`; add a library to the ecosystem file; never both.

`mise` also owns local runtime configuration, as `[env]` in three layers
that merge per key: `.mise.toml` (every variable, its dev default and its
documentation), `mise.test.toml` (what the test environment changes;
`make test` selects it with `MISE_ENV=test`) and gitignored
`mise.local.toml` (personal overrides and secrets, copied from
`mise.local.toml.example` by `gonext init`). There is no `.env`; the server
and the `gonext` CLI read real environment variables only. The CLI never
invokes `mise`: it needs `go` and `pnpm` on `PATH`, which `mise` provides.
Nothing from `mise` ships in a production image.

### 1. Install `mise`

On macOS (using Homebrew):
```bash
brew install mise
```
*(For other platforms, see the [official mise installation guide](https://mise.jdx.dev/getting-started.html)).*

### 2. Activate `mise` in Your Shell

Add `mise` activation to your shell profile if not already done. With the
hook in place, entering this directory puts the pinned tools and the `[env]`
variables in your shell — you never type `mise exec` yourself:

**Zsh (`~/.zshrc`):**
```bash
echo 'eval "$(mise activate zsh)"' >> ~/.zshrc
source ~/.zshrc
```

**Bash (`~/.bashrc`):**
```bash
echo 'eval "$(mise activate bash)"' >> ~/.bashrc
source ~/.bashrc
```

**oh-my-zsh (add it to your plugins definition in `~/.zshrc.`):**
```bash
plugins=(
    ...
    mise
    )
```

The `Makefile`, `lefthook.yml` and CI wrap every command in `mise exec --`
instead, because the shell hook never fires there (an agent's shell, a git
hook run by an IDE, a CI runner).

### 3. Install Project Toolchains

From the project root:
```bash
mise install
```

### 4. Install Git Hooks

From the project root, after `mise install`:
```bash
make hooks-install
```

This installs the `lefthook` pre-commit hook, which runs formatting, linting,
type-checking, dependency auditing, and secret scanning on every commit —
the same checks CI runs.
