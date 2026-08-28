---
name: new-branch
description: Use when starting new work and no feature branch exists yet - derives a descriptive <type>/<kebab-description> branch name from the task at hand and creates it, using this repo's branch type conventions (feat, fix, docs, chore).
---

# New Branch

Create a git branch with a name that reflects the actual work about to be done, instead of a generic or timestamped name.

## Steps

1. **Determine the work type.** Match the task to one of: `feat`, `fix`, `docs`, `chore`. If it's ambiguous, prefer `chore` for maintenance-style tasks and `feat` for new functionality.

2. **Derive a short description.** Summarize the task in 3-6 words, from the current conversation/task context — not from asking the user, unless intent is genuinely unclear. Convert to kebab-case: lowercase, hyphens between words, strip punctuation, no articles ("a", "the") unless meaningful.

3. **Compose the branch name** as `<type>/<description>`, e.g. `feat/user-auth-flow`, `fix/pagination-off-by-one`.

4. **Check for collisions:**
   ```bash
   git branch --list "<type>/<description>"
   git ls-remote --heads origin "<type>/<description>"
   ```
   If the name already exists, append `-2`, `-3`, etc.

5. **Check working tree state first.** Run `git status --short`. Uncommitted changes are carried onto the new branch by `git checkout -b` (this is usually what's wanted, not a problem to fix) — just don't discard anything.

6. **Create and switch:**
   ```bash
   git checkout -b <type>/<description>
   ```

7. **Confirm** the new branch name to the user in one line.

## Notes

- Don't invent a branch name before knowing what the task actually is — if invoked with no clear task context, ask the user briefly what the work is about.
- Keep the description specific enough to be meaningful in a branch list months later; avoid vague names like `fix/bug` or `feat/update`.
