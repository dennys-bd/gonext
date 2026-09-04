---
description: "Freeform ideation session: discuss feature ideas/insights, capture each as a GitHub issue, no spec produced."
argument-hint: "[optional starting idea or topic]"
---

# /gonext-ideate

A lightweight capture flow for open-ended brainstorming — talking through
feature ideas, insights, and rough concepts, and registering each as a
backlog item. **This is explicitly not the `brainstorming` skill and does
not produce a spec.** No design doc, no architecture, no approach
trade-offs. If a discussed idea needs that level of rigor later, that's a
separate `/gonext-flow` Stage 1 session — this command's output is
intentionally shallow: an issue that captures *what* and *why*, for later
triage.

**Input**: `$ARGUMENTS` — an optional idea to start with. If empty, open
with: "What's on your mind — what should gonext do that it doesn't
today?"

---

## Session shape

The conversation itself can happen in Portuguese, or whatever language the
user is using — match their language, don't force English in the chat.
**Every GitHub issue this command creates (title, body, checklist items)
must be written in English**, regardless of what language the discussion
was in — translate the captured idea, don't transcribe it. This matches
the rest of the repo (issues, `roadmap.md`, code, commit messages are all
English) and keeps the backlog consistent for anyone reading it later.

This is a free session, not one question per idea. The user may drop
several ideas in a row, jump between them, or think out loud. Track every
distinct idea mentioned during the conversation — don't let earlier ones
get lost when a new one comes up.

For **each** idea, before writing anything:

1. **Reflect it back in one sentence** — confirm you understood it, not as
   a formal restatement ritual, just enough to catch a misread.
2. **Ask ONE light clarifying question if genuinely needed** — only if the
   idea is too vague to write a useful issue (e.g. "better auth" — better
   how?). Do not interrogate. This is not the brainstorming skill's
   multi-question depth; one question, or none, per idea.
3. **Do not propose approaches, trade-offs, or architecture.** If the user
   starts going there themselves, that's fine — capture what they say —
   but don't drive toward implementation detail. Redirect back to "what
   problem / what capability" if the conversation drifts into a full
   design (and mention that `/gonext-flow` is the place for that once this
   idea is prioritized).

## Registering an idea

Once an idea is clear enough to write down (does not need to be fully
resolved — "rough but real" is enough), decide whether it's a **simple
issue** or an **epic** (see below), then create it via the GitHub MCP
connection (not `gh` CLI).

### Simple issue (default)

- **Title**: short, action-oriented, matches this repo's existing issue
  title conventions (see open issues for tone — e.g. "Gap: ...",
  "Tech debt: ...", or a plain feature-style title for net-new ideas).
- **Body**: 2-5 sentences capturing the idea, the problem/opportunity it
  addresses, and any context or constraint the user mentioned. Note it
  came out of an ideation session, not a scoped request. No sections, no
  template ceremony — this is a capture, not a PRD.
- **Label**: always add `idea`. Additionally infer and add one of
  `core-foundation` / `framework-gap` / `tech-debt` **only when the fit is
  clear** from what was discussed (matching this repo's existing label
  semantics — see `roadmap.md`'s label table). If it's ambiguous, leave it
  unlabeled beyond `idea` rather than guessing — mislabeling costs more
  triage time later than an unlabeled item does.

### Epic (multi-part ideas)

If an idea clearly bundles multiple independent pieces of work (e.g. "add
GraphQL support" implying schema generation + resolvers + client codegen +
docs, or anything the user describes as more than one deliverable), flag
it in the moment: "this sounds like it has a few separate parts — want me
to structure it as an epic with sub-issues, or keep it as one issue?" Do
not decide silently either way — this is the one judgment call this
command makes out loud, because getting it wrong either fragments a simple
idea or buries a real epic as one vague issue.

If the user confirms epic:

1. Create the **parent issue** exactly like a simple issue above, plus the
   label `epic` (in addition to `idea` and any inferred tier label). Body
   states the overall goal — not implementation detail — and ends with a
   checklist of the parts discussed so far (plain markdown checklist,
   `- [ ] <part>`), one line per anticipated sub-issue.
2. Create one **child issue** per part, same conventions as a simple
   issue, each scoped to that one piece.
3. Link every child to the parent using the GitHub MCP's native sub-issue
   tools (add-sub-issue / equivalent) — not ECC's `epic-*` coordination
   scripts (`scripts/github-coordination.js`). Those scripts maintain a
   separate SQLite cache and a coordination block inside the issue body
   for multi-worker claim/review orchestration — more machinery than this
   repo needs; native GitHub sub-issues are the parent/child link the
   Projects board will read natively once it's finished.
4. If new parts come up later in the same ideation session that clearly
   belong to an already-created epic, add them as new child issues linked
   to that epic rather than starting a new parent.

In both cases:

- Confirm the created issue number(s) and title(s) back to the user in one
  line (for an epic: parent number + each child number), then continue the
  session for the next idea.

## Closing the session

When the user signals they're done (explicitly, or the conversation
naturally winds down), summarize: list every issue created this session
(`#N — title`) in one short list. Don't re-pitch or re-discuss them.

## Explicitly Out of Scope

- No spec file, no `docs/superpowers/specs/` entry — that only happens via
  `/gonext-flow` Stage 1 once an idea is picked up for real work.
- No plan, no branch, no code. This command never leaves the conversation.
- No priority ranking beyond the best-effort label inference above —
  ordering the backlog is a separate activity.
- `gh` CLI — use the GitHub MCP connection for issue creation, consistent
  with `/gonext-flow`.

## First-run setup

The `idea` and `epic` labels do not exist in this repo yet (as of this
command's creation). Before the first issue is created, check whether
`idea` exists; if not, create both `idea` and `epic` via the GitHub MCP
(match the existing label style — short description, a color distinct
from the tier labels `core-foundation`/`framework-gap`/`tech-debt`). Do
this silently as part of registering the first idea, not as a separate
announced step.
