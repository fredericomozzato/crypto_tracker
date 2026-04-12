---
branch: feat/008-list-mode-edit-delete-holding
revision: 1
status: done
---

# Slice 008 — List Mode + Edit + Delete Holding (Revision 1)

## Smoke test + completeness audit

No findings. All scope items implemented, test coverage adequate, verification
commands satisfied.

## Feature review

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| F1 | MED | FIXED | Adding a holding from listing mode returns to browsing instead of listing |

**F1** `internal/ui/portfolio.go:319`, `internal/ui/portfolio.go:958`

When the user presses `a` in listing mode, `cmdOpenCoinPickerFromList(mode)` is
called. The function accepts a `listing` parameter but never uses it — the
comment says "preserving list mode for return" but the parameter is dead code.
The `coinPickerReadyMsg` handler overwrites `m.mode` to `addCoin{}` without
tracking the parent mode. After the add-holding flow completes,
`holdingsSavedMsg` transitions to `browsing{}` because `m.mode` is `addAmount`,
not `editingAmount`. Pressing Esc from `addCoin` also always returns to
`browsing{}`, losing the listing state. The fix is to add a parent-mode tracker
(e.g. an `origin portfolioMode` field) to `addCoin`/`addAmount` so they can
return to the correct mode.

## Implementation review

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| I1 | LOW | DISCARDED | Missing Ctrl+F/Ctrl+B key bindings for preview scrolling per PRD |
| I2 | LOW | FIXED | Listing mode status bar omits g/G and q shortcuts |

**I1** `internal/ui/portfolio.go:171-178`

DISCARDED - DO NOT FIX

**I2** `internal/ui/portfolio.go:759`

The listing mode status bar reads `j/k holdings • Enter edit • X delete • a add • Esc menu`.
The PRD specifies: `j/k holdings • g/G top/bottom • Enter edit • X delete • a add holding • Esc back to menu • q quit`.
The implementation omits `g/G top/bottom` and `q quit`. The `g` and `G` keys
are handled in listing mode but not advertised in the status bar.
