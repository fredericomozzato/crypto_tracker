---
branch: feat/006-create-portfolio-left-panel
revision: 3
status: done
---

# Slice 6 — Create portfolio + left panel (Revision 3)

## Smoke test + completeness audit

No findings. Build passes cleanly, all tests pass. This revision captures two bugs
discovered via manual inspection after revision 2 fixes were applied.

## Implementation review

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| I1 | HIGH | FIXED | `portfoliosLoadedMsg` is dropped when Markets tab is active on startup |
| I2 | LOW  | FIXED | Left and right panels have no titles ("Portfolios" / "Holdings") |

**I1** `internal/ui/app.go:102-111`  
`app.go` routes all non-WindowSize messages exclusively to the *active* child model.
`portfolio.Init()` fires `cmdLoadPortfolios()`, which returns `portfoliosLoadedMsg`
asynchronously. Since the app starts on the Markets tab, that message arrives while
`tabMarkets` is active and is dispatched to `markets.update()` — which ignores it.
The portfolio model never receives the response, so `m.portfolios` stays empty. The
user sees "no portfolios — press n to create one" even when portfolios exist in the
database. Creating a new portfolio triggers a fresh load (`cmdCreatePortfolio` calls
`GetAllPortfolios` and returns another `portfoliosLoadedMsg`) which arrives while the
Portfolio tab is now active and is correctly processed — which is why all previously
created portfolios appear after the first creation.

Fix: background messages (any message that is not a `tea.KeyMsg` or
`tea.WindowSizeMsg`) should be fanned out to both children, not just the active one.
The simplest change is to restructure the fallthrough routing in `app.go` so that
data messages go to both models:

```go
// After the WindowSizeMsg / KeyMsg cases:
var cmd1, cmd2 tea.Cmd
m.markets, cmd1 = m.markets.update(msg)
m.portfolio, cmd2 = m.portfolio.update(msg)
return m, tea.Batch(cmd1, cmd2)
```

Key presses must still go only to the active tab (already handled above the switch),
so this fan-out only affects data/response messages that fall through to the bottom
of `Update`.

**I2** `internal/ui/portfolio.go` — `renderLeftPanel` / `renderRightPanel`  
Neither panel displays a title. The left panel should have a "Portfolios" header and
the right panel a "Holdings" header, so the user can identify each area at a glance
(matching standard two-panel TUI conventions and the PRD's layout description).

Fix: prepend a bold or underlined title line to each panel's rendered content, inside
the border. The title should count against `contentHeight` so the border and status
bar math stays correct (i.e. portfolio rows = `contentHeight - 1` after reserving one
line for the title). Example:

```go
// renderLeftPanel
titleStyle := lipgloss.NewStyle().Bold(true)
title := titleStyle.Render("Portfolios")
// ... render portfolio rows using (height - 1) remaining lines
return title + "\n" + rows
```

Apply the same pattern to `renderRightPanel` with the label `"Holdings"`.
