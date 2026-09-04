---
description: "End-to-end gonext dev flow: brainstorm → plan → branch → TDD implementation → review → security → PR."
argument-hint: "[idea, or a GitHub issue/PR link you paste in]"
---

# /gonext-flow

Orchestrates this repo's development pipeline end to end, stopping at every
human approval gate. Each stage below is a delegation to an existing skill,
agent, or command — this file only sequences them and decides routing
(backend vs frontend vs both) based on files actually touched.

**Input**: `$ARGUMENTS` — a feature idea in free text, or a GitHub issue/PR
link the user pastes in when invoking this command (they bring the link;
this command never guesses or searches one up). If empty, ask what to work
on.

---

## Model Routing

Each stage below uses the model its agent/skill already declares by
default — no override needed unless the user asks for one explicitly
(`/model <name>` mid-session, or `--model` on a `claude -p` invocation).
Recorded here so routing is visible and intentional, not incidental:

| Stage | Actor | Model | Why |
|---|---|---|---|
| 1 — Spec | `superpowers:brainstorming` (inline, main session) | whatever model the session is running | interactive dialogue with the user, no agent dispatch |
| 2 — Plan | `/plan` inline, or `ecc:planner` agent | `opus` (`ecc:planner`'s declared default) | planning quality matters most where mistakes are expensive to unwind |
| 3 — Branch | `new-branch` skill (inline) | session model | trivial, no dispatch |
| 4 — Implementation | `ecc:tdd-guide` | `sonnet` | high-volume, well-specified work once the plan is approved |
| 4 — Build-fix (Go) | `ecc:go-build-resolver` | `sonnet` | mechanical error resolution |
| 4 — Build-fix (frontend) | `ecc:react-build-resolver` | `sonnet` | mechanical error resolution |
| 5 — Code Review (Go) | `ecc:go-reviewer` | `sonnet` | ECC's own default for this agent |
| 5 — Code Review (frontend) | `ecc:typescript-reviewer` + `ecc:react-reviewer` | `sonnet` | ECC's own default for these agents |
| 5 — E2E | `ecc:e2e-runner` | `sonnet` | ECC's own default |
| 6 — Security | `ecc:security-reviewer` (via `/security-scan`) | `sonnet` | ECC's own default; escalate to `opus` by hand for a security-sensitive surface (auth, payments, crypto) |
| 7 — PR | `/pr` inline (no subagent) | session model | mechanical push/template-fill/gh-metadata work |

If a stage's actual risk doesn't match this table for a given task (e.g. a
`tdd-guide` pass touching `auth/` — genuinely security-sensitive, not just
mechanical), say so and bump that one stage to `opus` rather than silently
using the default.

## Stage 1 — Spec (human-in-the-loop, brainstorming only)

Invoke the `superpowers:brainstorming` skill — **only this piece of
superpowers is in scope for this repo**, nothing else from that plugin.

- Classify spike / bounded / architectural per that skill's rules.
- All questions, approach trade-offs, and design go through chat with the
  user — this is the one stage the user actively drives.
- Architectural path writes `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md`
  and commits it (matches this repo's existing spec tree — do not redirect
  specs to the GitHub project board; the board tracks status, not content).
- Bounded path: short in-chat design, explicit approval, no spec file.
- **HARD GATE**: do not proceed to Stage 2 without explicit user approval
  of the design/spec.

## Stage 2 — Plan

Use ECC's `/plan` command (or the `ecc:planner` agent directly for complex
architecture) to produce `.claude/plans/<topic>.plan.md`, following this
repo's existing plan format (see `.claude/plans/backend-live-reload.plan.md`
for the convention: Summary, Patterns to Mirror, Files to Change, Tasks,
Risks) — `/plan`'s own PRD-artifact output structure is compatible with
this; use the repo's existing headings when they diverge.

For bounded-path work with no spec file, skip straight to Stage 3 after the
Stage 1 in-chat approval — no plan document needed.

**GATE**: present the plan, wait for explicit approval before Stage 3.

## Stage 3 — Branch

Invoke the `new-branch` skill (`.claude/skills/new-branch/SKILL.md`) to
create a correctly named `feat/`, `fix/`, `docs/`, or `chore/` branch before
any file changes.

## Stage 4 — Implementation (TDD)

Use the `ecc:tdd-guide` agent to work the plan task by task
(RED → GREEN → refactor → coverage check), producing
`.claude/tdd/<topic>.tdd.md` matching this repo's existing evidence-report
convention (see `.claude/tdd/backend-live-reload.tdd.md`: user journeys,
deviations from the plan, task-by-task report, RED/GREEN evidence).

Route build-error resolution by the area touched:

| Area touched | Build-fix agent |
|---|---|
| `backend/`, `internal/`, `cmd/`, `auth/` (Go) | `ecc:go-build-resolver` |
| `frontend/` (Next.js/TS/React) | `ecc:react-build-resolver` |
| Both | run both, independently, in parallel |

Standard validation commands (from `CLAUDE.md` / `make check`):
```sh
go build ./auth/... ./cmd/... ./internal/... .
go vet ./auth/... ./cmd/... ./internal/... .
go test -race ./auth/... ./cmd/... ./internal/... .
```
plus the frontend project's `pnpm typecheck` / `pnpm lint` / `pnpm test` as
applicable. If `templates/` changed, run `make golden` and confirm
`git status --short golden/` is clean before moving on (drift guarantee).

## Stage 5 — Code Review (separate pass, never self-approval)

Never run this in the same context that wrote the code. Route by area:

| Area touched | Reviewer agent(s) |
|---|---|
| Go (`backend/`, `internal/`, `cmd/`, `auth/`) | `ecc:go-reviewer` |
| Frontend (Next.js/TS/React) | `ecc:typescript-reviewer` + `ecc:react-reviewer` |
| Both | run each reviewer independently, in parallel |
| User-facing flow changed | also run `ecc:e2e-runner` (Playwright) |

Address CRITICAL/HIGH findings before Stage 6. Re-review after fixes if
CRITICAL findings required non-trivial changes.

## Stage 6 — Security Review

Run `/security-scan` (ECC, backed by `ecc:security-reviewer` /
AgentShield). Fix CRITICAL/HIGH findings; re-run to confirm before Stage 7.

## Stage 7 — PR

Run `/pr` (ECC) to push, discover the PR template, and open the pull
request against `main`, referencing the spec/plan/TDD-evidence artifacts
from Stages 1, 2, and 4.

After the PR is opened:
- Close the tracked issue from the PR body (`Closes #N`). Status lives on
  the project board, never in a file — do not add status markers to
  `roadmap.md`.
- Only touch `roadmap.md` if the work changed a *design* decision (what a
  component is or why), not merely its completion state.
- Issue and board operations go through the GitHub MCP connection, not the
  `gh` CLI.

---

## Gates Summary (never skip)

1. Spec/design approved by user (Stage 1) — HARD GATE, no exceptions.
2. Plan approved by user (Stage 2, architectural path only).
3. Code review clean of CRITICAL/HIGH (Stage 5) before security scan.
4. Security scan clean of CRITICAL/HIGH (Stage 6) before PR.

## Explicitly Out of Scope

- Any superpowers skill other than `brainstorming` (no `subagent-driven-development`,
  no visual companion by default, etc.) — this repo only opts into the
  spec-writing piece of that plugin.
- `gh` CLI for GitHub operations — this repo's Claude Code is wired to
  GitHub via the `github` MCP plugin; prefer that over shelling out to `gh`
  when either path is available.
