# [PROJECT-NAME]

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
go run ./cmd/server
```

In another terminal:

```bash
curl localhost:8080/healthz
```

## Quick start (frontend)

```bash
cd frontend
pnpm install
pnpm dev
```

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
