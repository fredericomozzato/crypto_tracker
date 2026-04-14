---
branch: feat/010-docs-tooling-cleanup
revision: 1
status: done
---

# Slice 010 — Docs & tooling cleanup (Revision 1)

## Smoke test + completeness audit

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| F1 | HIGH | FIXED | CLAUDE.md not updated — scope items 1 & 2 specify CLAUDE.md but changes applied only to AGENTS.md |

**F1** `CLAUDE.md`

The issue scope explicitly references `CLAUDE.md` for removing the terminal size guard and
`gosimple` mentions. However, the changes were applied only to `AGENTS.md`. Both files exist
in the repo root and contain near-identical guidance content. `CLAUDE.md` still has:

- Line 81: `"Terminal too small — resize to at least 100×30"` sentence (terminal size guard)
- Line 243: ``- `gosimple` — suggests simpler constructs`` (active linters list)
- Line 320: `- gosimple` (golangci-lint config example)

These three stale entries must also be removed from `CLAUDE.md` to complete the scope. The
`Makefile` change (scope item 3) is correctly applied.

## Implementation review

No findings. The changes to AGENTS.md and Makefile are correct and minimal.