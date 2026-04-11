---
name: build
description: Use when the user runs /build in the crypto_tracker project to implement the IN_PROGRESS slice from the roadmap.
---

# Build — Slice Implementation

## Overview

Pick up the IN_PROGRESS slice, validate the plan against the current codebase, switch to
the feature branch, implement with TDD, and hand off to the user for manual testing.

**Announce at start:** "Using the build skill to implement the current IN_PROGRESS slice."

## Workflow

```dot
digraph build {
    "Load context" -> "Find IN_PROGRESS slice";
    "Find IN_PROGRESS slice" -> "Read issue plan";
    "Read issue plan" -> "Survey codebase";
    "Survey codebase" -> "Plan blockers?" [label="validate"];
    "Plan blockers?" -> "Prompt user, wait for resolution" [label="yes"];
    "Prompt user, wait for resolution" -> "Survey codebase";
    "Plan blockers?" -> "Checkout branch" [label="no"];
    "Checkout branch" -> "Branch already exists?" [label="check"];
    "Branch already exists?" -> "git checkout branch" [label="yes"];
    "Branch already exists?" -> "git checkout -b branch" [label="no"];
    "git checkout branch" -> "Implement (TDD)";
    "git checkout -b branch" -> "Implement (TDD)";
    "Implement (TDD)" -> "make check";
    "make check" -> "Failures?" [label="assess"];
    "Failures?" -> "Fix failures" [label="yes"];
    "Fix failures" -> "make check";
    "Failures?" -> "Set STATUS: IN_REVIEW + Prompt user to test" [label="no"];
}
```

## Steps

### 1. Load context

Read these two files before doing anything:
- `docs/PRD.md` — feature specs, UI layout, keyboard map, data model
- `docs/ARCHITECTURE.md` — conventions every file must follow

### 2. Find the IN_PROGRESS slice

Read `docs/roadmap.md`. Find the **single** section with `STATUS: IN_PROGRESS`.
Note the slice number, name, branch name, and any **IMPORTANT** constraint blocks.

If no slice is `IN_PROGRESS`, stop and tell the user to run `/refine` first.

### 3. Read the issue plan

Open `docs/issues/NNN-kebab-case-name.md` (matching the IN_PROGRESS slice).
Read the **entire** file — context, scope, file-by-file plan, implementation order, and verification commands.

### 4. Survey the codebase

Read every file the plan mentions as "modify". Read the existing test files to understand
established patterns. Read public interfaces the slice must satisfy.

Goal: confirm the plan's type signatures, import paths, and function names are still
accurate given the current code.

### 5. Validate the plan

Before touching any code, check for:

- **Blockers** — plan references a type, function, or interface that doesn't exist yet
- **Divergence** — current code has drifted from the plan's assumptions (renamed fields, changed signatures, etc.)
- **Ambiguity** — a step in the implementation order is too vague to execute safely

If any issue is found, describe it clearly and ask the user how to resolve it.
**Do not begin implementation until the plan is fully understood and sound.**

### 6. Checkout the feature branch

The branch name is in the issue frontmatter: `branch: feat/NNN-kebab-case-name`.

```bash
# Check current branch — must NOT be main
git branch --show-current

# Switch (create if it doesn't exist yet)
git checkout feat/NNN-kebab-case-name 2>/dev/null || git checkout -b feat/NNN-kebab-case-name
```

**Never implement on `main`.** If already on the correct feature branch, proceed.

### 7. Implement with TDD

Follow the plan's **Implementation order** section exactly — step by step, in the numbered
sequence. Do not reorder steps.

**Red-green cycle per step:**

1. Write the test(s) for the step.
2. Run `go test ./...` — confirm they fail (red).
3. Write the minimum implementation to make them pass.
4. Run `go test ./...` — confirm they pass (green).

**Commit at each logical checkpoint** (typically after each numbered step passes):

```bash
git add <specific files>
git commit -m "feat(slice-NNN): <short description of step>"
```

Keep commits atomic — one logical change per commit.

**Architecture non-negotiables** (from CLAUDE.md / ARCHITECTURE.md):
- `ctx context.Context` is the first parameter of every I/O function
- UI layer depends on `store.Store` interface, never on `*sql.DB`
- All side effects returned as `tea.Cmd`; no goroutines inside handlers
- `url.Values` for all query strings — no string interpolation
- Error wrapping: `fmt.Errorf("outer: %w", err)`
- Tests: real SQLite via `t.TempDir()`; `httptest.NewServer` for API fakes

### 8. Final quality check

Once all steps are implemented:

```bash
make check   # fmt + lint + test + vuln — ALL must pass
make build   # binary must compile cleanly
```

If anything fails, fix it before proceeding. Do not declare the build complete with
failing checks.

### 9. Update roadmap and issue

1. In `docs/roadmap.md`, change the slice's `STATUS: IN_PROGRESS` → `STATUS: IN_REVIEW`.
2. In the issue file frontmatter, change `status: in_progress` → `status: in_review`.

**Do NOT set `STATUS: DONE`.** The `done` status is only set by an explicit user instruction or a dedicated review agent after manual verification passes.

### 10. Prompt the user to test

Present a brief handoff message:

```
Implementation complete. Here's what was built:

<2–4 bullet summary of what was implemented>

To test manually:
<copy the Verification commands from the issue file>

All checks pass (make check + make build). Ready for your review.
```

Do NOT create a PR or merge. Hand off to the user for manual verification.
