---
branch: feat/005-tab-bar-empty-portfolio-tab
status: done
revision: 1
---

# Slice 5 — Tab bar + empty Portfolio tab

## Smoke test + completeness audit

`make check` passes clean: gofumpt, golangci-lint (0 issues), all tests (race detector clean),
govulncheck (no vulnerabilities). Binary builds successfully.

Smoke test via TTY unavailable on this runner, but `go build` completes without error.

All scope items from the issue are implemented:

- Two tabs rendered at the top ✓ — `renderTabBar()` produces styled `Markets` / `Portfolio` labels
- `Tab` / `Shift+Tab` / `1` / `2` switching ✓ — `AppModel.Update` handles all four
- Portfolio empty state ✓ — `PortfolioModel.View()` returns `"no portfolios — press n to create one"`
- `InputActive()` suppression wired ✓ — `activeInputActive()` delegates to active child; integration test deferred to Slice 6 per plan
- All 40 markets tests, 17 app tests, and 6 portfolio tests from the issue plan are present and passing

No gaps found.

---

## Implementation review

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| I1 | LOW | FIXED | `MarketsModel.View()` checks terminal size — plan said not to |
| I2 | LOW | FIXED | Duplicate context.Background() vars in test package |

**I1** `internal/ui/markets.go:201–203`  
`MarketsModel.View()` returns the "Terminal too small" message when dimensions are below
100×30. The issue plan explicitly states: *"View does NOT check terminal size — AppModel
guards that."* `AppModel.View()` already has the guard, so this is redundant. It adds
coupling to the terminal-size constraint in a child model that shouldn't care about it.
Fix: remove the guard from `MarketsModel.View()`.

**I2** `internal/ui/testhelpers_test.go:87` and `internal/ui/markets_test.go:14`  
Two separate `context.Background()` variables exist in the same test package:
`var testCtx` (testhelpers_test.go) used only by `setupMarketsModel`, and
`var marketsTestCtx` (markets_test.go) used throughout markets tests.
They are identical in value. Fix: remove `marketsTestCtx` from markets_test.go and
replace all uses with `testCtx`.
