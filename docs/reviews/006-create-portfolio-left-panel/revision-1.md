---
branch: feat/006-create-portfolio-left-panel
revision: 1
status: done
---

# Slice 6 — Create portfolio + left panel (Revision 1)

## Smoke test + completeness audit

Build passes cleanly. All 110 tests pass with race detector. Verification commands
from the issue file confirmed:

- `go test -race -v ./internal/store/... -run "Portfolio"` → 5 tests PASS
- `go test -race -v ./internal/ui/... -run "Portfolio"` → 23 tests PASS

All scope items are implemented:
- Left panel with `▶` cursor and `j`/`k` navigation ✓
- `n` opens create dialog (max 50 chars, `Enter` saves, `Esc` cancels) ✓
- After creation: cursor moves to new portfolio, mode returns to browsing ✓
- Store tests: `TestCreatePortfolio`, `TestCreatePortfolioSetsCreatedAt`,
  `TestGetAllPortfoliosEmpty`, `TestGetAllPortfoliosMultiple`,
  `TestCreatePortfolioReturnsInsertedID` all present and passing ✓
- UI dialog state transition tests: all 17 required tests present and passing ✓

No completeness gaps found.

## Implementation review

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| I1 | MED | FIXED | `renderDialogOverlay` receives `background` but never uses it — panels not shown behind dialog |
| I2 | LOW | FIXED | `renderRightPanel` has unused `height int` parameter |

**I1** `internal/ui/portfolio.go:227-250`  
`renderDialogOverlay(background string)` accepts the rendered panels view as `background`
but the function body never references it. `lipgloss.Place` is called with only `m.width`,
`m.height`, the dialog widget, and a hardcoded black whitespace fill — the `background`
string is silently dropped. The result is that when the create dialog is open, the two
panels disappear and are replaced by a solid black background.

Both the PRD and the issue plan specify: "a centered overlay dialog is rendered on **top
of the panels**" / "The background panels are still rendered behind them." The current
implementation does not satisfy this.

Fix: use `background` as the canvas. One approach is to render the dialog into the
background using ANSI-aware overlay (e.g. `lipgloss.Place` does not overlay — it
constructs a fresh block). A simpler approach accepted by the project is to render the
dialog on top of the existing string using `lipgloss.Place` with the background string
passed as the base. Check the lipgloss v1.x API for `PlaceOverlay` or equivalent. If
no overlay API is available, document the limitation and treat the black background as a
known simplification.

**I2** `internal/ui/portfolio.go:204`  
`renderRightPanel` is declared as `func (m PortfolioModel) renderRightPanel(height int) string`
but `height` is never read inside the function body. It returns a static string in all
branches. The parameter is dead code and misleads Slice 7 implementers, who will add the
holdings table and need `height` for scrolling — they may assume the parameter is already
wired correctly when it is not.

Fix: remove the `height` parameter for now (it has no effect) and add it back when the
holdings table rendering requires it in Slice 7.
