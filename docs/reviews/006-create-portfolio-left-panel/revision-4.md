---
branch: feat/006-create-portfolio-left-panel
revision: 4
status: done
---

# Slice 6 — Create portfolio + left panel (Revision 4)

## Smoke test + completeness audit

No findings. Build passes cleanly, all 111 tests pass with race detector. All scope items
remain implemented and test coverage is adequate.

- `go test -race -v ./internal/store/... -run "Portfolio"` → 5 tests PASS
- `go test -race -v ./internal/ui/... -run "Portfolio"` → 23 tests PASS
- `golangci-lint run ./...` → no issues

## Implementation review

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| I1 | LOW | FIXED | `renderLeftPanel` receives `contentHeight-2` — border offset already handled by lipgloss, wastes 2 panel rows |

**I1** `internal/ui/portfolio.go:171`  
`m.renderLeftPanel(contentHeight-2, leftPanelInner)` is called with `// -2 for border`
as the comment. However, the border height is already accounted for by
`leftStyle.Height(contentHeight)` — lipgloss adds the 2 border rows on top of the inner
content height automatically. Subtracting 2 from the argument passed into `renderLeftPanel`
means the function computes `contentHeight := height - 1 = contentHeight_view - 3`, limiting
the visible portfolio list to 2 fewer rows than the panel can actually display.

At minimum terminal size (30 rows), the portfolio model height is 29, `contentHeight_view`
is 26, and the panel can display 25 portfolios (26 inner rows minus 1 title row). The
current code caps visible entries at 23 instead. For users with more than 23 portfolios at
minimum terminal size, the last 2 entries are hidden without any scroll indicator.

Fix: pass `contentHeight` directly (not `contentHeight-2`):

```go
leftContent := m.renderLeftPanel(contentHeight, leftPanelInner)
```

Inside `renderLeftPanel` the `height` parameter will now equal the panel's inner content
height (`contentHeight_view`), and `contentHeight := height - 1` correctly yields the
number of portfolio rows after reserving 1 line for the title.
