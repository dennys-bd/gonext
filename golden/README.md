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

`docs/openapi.yaml` is produced from the backend's route registrations with no database or environment needed:

```bash
make openapi         # regenerate docs/openapi.yaml and frontend/lib/api/
make openapi-check   # fail if docs/openapi.yaml is stale (CI runs this)
```

If `frontend/lib/api/` is missing (it is gitignored), `pnpm --dir frontend codegen` regenerates it.

## Development Prerequisites

This repository uses [**`mise`**](https://mise.jdx.dev/) to manage runtime toolchains and developer environments (such as Go versions).

### 1. Install `mise`

On macOS (using Homebrew):
```bash
brew install mise
```
*(For other platforms, see the [official mise installation guide](https://mise.jdx.dev/getting-started.html)).*

### 2. Activate `mise` in Your Shell (If you installed using brew)

Add `mise` activation to your shell profile if not already done:

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
