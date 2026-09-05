---
description: "End-to-end gonext dev flow: brainstorm → plan → branch → TDD implementation → review → security → PR."
argument-hint: "[idea, a GitHub issue/PR link, or a path to an existing spec file]"
---

# /gonext-flow

Orchestrates this repo's development pipeline end to end, stopping at every
human approval gate. Each stage below is a delegation to an existing skill,
agent, or command — this file only sequences them and decides routing
(backend vs frontend vs both) based on files actually touched.

**Input**: `$ARGUMENTS` — one of:

- a feature idea in free text;
- a GitHub issue/PR link the user pastes in (they bring the link; this
  command never guesses or searches one up);
- **a path to an existing spec file** (typically
  `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md`).

If empty, ask what to work on.

**Entry point**: when the argument is a path to an existing spec file, when
the user says the spec is already written, or when the argument is an issue
whose board card is already in **Ready** — **skip Stage 1 entirely and start
at Stage 2**. The spec is the output of Stage 1; handed one, that
stage has nothing left to do. Before proceeding:

1. Read the spec in full and restate its scope in two or three sentences.
2. Confirm with the user that the spec is current and approved
   (this replaces the Stage 1 hard gate — it is not waived, just already
   satisfied by an approved document).
3. Find the tracked issue for it and move its board card to **In Progress**
   per *Board status* below.

Do not re-run brainstorming over an approved spec; if reading it surfaces a
genuine gap, raise the specific gap and ask whether to amend the spec file
or proceed with a stated assumption.

### Resolving the spec for an issue already in Ready

**Ready** means the spec is approved (see *Board status*), so an issue in
that column must have one. Find it in this order:

1. **The issue's comments.** Look for the `spec: docs/superpowers/specs/<file>.md`
   comment the spec convention posts (`issue_read` with `get_comments`).
   That pointer wins over anything inferred.
2. **Filename match.** Failing a comment, glob
   `docs/superpowers/specs/*-design.md` and match the topic slug against the
   issue title (and the current branch name, if one already exists for this
   work). Accept a match only if it is unambiguous.

**If neither resolves — or the comment names a path that is not on disk —
STOP.** Do not fall back to Stage 1 and brainstorm a replacement, and do not
proceed to Stage 2 without a spec. Report the issue number, the column it is
in, and what was searched, then ask the user whether to:

- run Stage 1 to write the missing spec (and move the card back), or
- point at the right spec file, or
- fix the stale `spec:` comment on the issue.

A card in **Ready** with no findable spec is a bookkeeping error worth
surfacing, not a gap to paper over.

**Reading the column**: the GitHub MCP connection cannot read Projects v2
fields any more than it can write them (see *Board status*). If `gh project
item-list` is unavailable for lack of the `project` scope, treat the presence
of a `spec:` comment on the issue as the signal that Stage 1 is already done,
and say that is what you are going on.

---

## Model Routing

Each stage below uses the model its agent/skill already declares by
default — no override needed unless the user asks for one explicitly
(`/model <name>` mid-session, or `--model` on a `claude -p` invocation).
Recorded here so routing is visible and intentional, not incidental:

| Stage | Actor | Model | Why |
|---|---|---|---|
| 1 — Spec | `superpowers:brainstorming` (inline, main session) | whatever model the session is running | interactive dialogue with the user, no agent dispatch |
| 2 — Plan | `ecc:planner` agent | `opus` (`ecc:planner`'s declared default) | planning quality matters most where mistakes are expensive to unwind |
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

## Board status (do this at every stage boundary)

Status for this repo lives on the [project board](https://github.com/users/dennys-bd/projects/6),
never in a file. The card for the tracked issue moves as the flow advances —
this is part of the flow, not an afterthought at the end:

| When | Column |
|---|---|
| Spec approved (end of Stage 1), or an already-approved spec handed in | **Ready** |
| Branch created and implementation started (Stage 3 → 4) | **In Progress** |
| PR opened (end of Stage 7) | **In Review** |
| PR merged | **Done** |

Rules:

- Announce each move in chat as it happens (`board: #N → In Progress`), so
  the user can see the card is tracking reality.
- If no issue exists for the work, create one before Stage 3 and put it in
  **Ready** — the flow always has a card to move.
- Never batch the moves at the end. A card sitting in **Ready** while the
  code is written is the failure this section exists to prevent.
- Do not close the issue by hand; `Closes #N` in the PR body does it, and
  merging is what drives **Done**.

**Tooling caveat**: the GitHub MCP connection covers issues but not
Projects v2 fields, so the column change is the one GitHub operation that
goes through the `gh` CLI:

```sh
gh project item-list 6 --owner dennys-bd --format json   # find the item id
gh project item-edit --id <item-id> --field-id <status-field-id> \
  --project-id <project-id> --single-select-option-id <option-id>
```

This needs the `project` scope on the token (`gh auth refresh -s project`).
If the scope is missing, say so once and ask the user to move the card
themselves, naming the exact column — never silently skip the update.

## Stage 1 — Spec (human-in-the-loop, brainstorming only)

Invoke the `superpowers:brainstorming` skill — **only this piece of
superpowers is in scope for this repo**, nothing else from that plugin.

- Classify spike / bounded / architectural per that skill's rules.
- All questions, approach trade-offs, and design go through chat with the
  user — this is the one stage the user actively drives.
- Architectural path writes `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md`
  (matches this repo's existing spec tree — do not redirect specs to the
  GitHub project board; the board tracks status, not content). That tree is
  gitignored, so the file is **not** committed.
- Once the spec file is written, comment its path on the tracked issue
  (`spec: docs/superpowers/specs/<file>.md`) via the GitHub MCP connection,
  so the issue carries a pointer to where the design lives. The path is a
  local breadcrumb, not a link anyone else can open — that is understood
  and intended.
- Bounded path: short in-chat design, explicit approval, no spec file.
- **HARD GATE**: do not proceed to Stage 2 without explicit user approval
  of the design/spec.
- On approval, move the card to **Ready**.

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

Once the branch exists, move the card to **In Progress**.

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
- Move the card to **In Review**.
- Close the tracked issue from the PR body (`Closes #N`). Status lives on
  the project board, never in a file — do not add status markers to
  `roadmap.md`. The card reaches **Done** when the PR merges.
- Only touch `roadmap.md` if the work changed a *design* decision (what a
  component is or why), not merely its completion state.
- Issue and board operations go through the GitHub MCP connection, not the
  `gh` CLI.

---

## Gates Summary (never skip)

1. Spec/design approved by user (Stage 1) — HARD GATE, no exceptions.
   Satisfied up front when the flow is entered with an approved spec file.
2. Plan approved by user (Stage 2, architectural path only).
3. Code review clean of CRITICAL/HIGH (Stage 5) before security scan.
4. Security scan clean of CRITICAL/HIGH (Stage 6) before PR.

## Explicitly Out of Scope

- Any superpowers skill other than `brainstorming` (no `subagent-driven-development`,
  no visual companion by default, etc.) — this repo only opts into the
  spec-writing piece of that plugin.
- `gh` CLI for GitHub operations — this repo's Claude Code is wired to
  GitHub via the `github` MCP plugin; prefer that over shelling out to `gh`
  when either path is available. The single exception is moving a project
  board card, which the MCP connection cannot do (see *Board status*).
