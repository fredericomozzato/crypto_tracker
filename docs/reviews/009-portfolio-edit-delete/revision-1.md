---
branch: feat/009-portfolio-edit-delete
revision: 1
status: done
---

# Slice 9 — Portfolio Management: Edit and Delete (Revision 1)

## Smoke test + completeness audit

All 220 tests pass. Binary compiles cleanly (`go build ./...`). No panics.

**Completeness gaps:**

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| C1 | LOW | FIXED | `TestUniquePortfolioName` missing from `internal/db/db_test.go` |

**C1** `internal/db/db_test.go`
The issue plan specifies adding `TestUniquePortfolioName` to `internal/db/db_test.go` to verify the schema-level UNIQUE constraint. The file still contains only the stub `TestPackageCompiles`. Equivalent coverage exists at the store layer (`TestCreatePortfolioDuplicateName` in `sqlite_test.go`), so this is not a coverage gap — it is an unexecuted plan step. Worth adding for completeness at the DB layer.

---

## Implementation review + user-reported findings

The following findings come from both code review and manual testing by the developer.

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| F1 | HIGH | FIXED | Duplicate name error on create shows raw wrapped Go error, unstyled |
| F2 | HIGH | FIXED | Edit/create dialogs grow as text is typed — no fixed width |
| F3 | HIGH | FIXED | `a` (add holding) accessible from browsing mode — should be list-mode only |
| F4 | MED  | FIXED | Enter on empty portfolio is a no-op — should enter holdings panel |

---

**F1** `internal/ui/portfolio.go:928–940` — `cmdCreatePortfolio`

When `CreatePortfolio` hits the UNIQUE constraint, SQLite returns an error. `cmdCreatePortfolio` wraps it and sends an `errMsg`. The `errMsg` handler sets `m.lastErr = msg.err.Error()`, which results in a string like `creating portfolio "Foo": UNIQUE constraint failed: portfolios.name`. This is displayed in `renderStatusBar` as plain text with no colour change, appended to the full key-hint line — easy to miss.

The fix is two-part:
1. Intercept the UNIQUE constraint error in `cmdCreatePortfolio` (check `strings.Contains(err.Error(), "UNIQUE")`) and return a friendly `errMsg{err: errors.New("portfolio \"Foo\" already exists")}`.
2. Make `renderStatusBar` render the error segment in red (consistent with how the inline dialog errors are styled): use `lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("error: " + m.lastErr)` for the error suffix.

Note: the `editingPortfolio` mode pre-validates duplicates client-side and shows a red inline message in the dialog — that path already works well. This finding is specific to the `creating` path, where validation goes through the DB layer.

---

**F2** `internal/ui/portfolio.go:1196–1207` — `renderEditPortfolioDialog`; also `internal/ui/portfolio.go:606–611` — `creating` dialog in `View()`

Neither the edit-portfolio dialog nor the create-portfolio dialog applies a fixed `Width()` to its outer `lipgloss.NewStyle()`. Lipgloss sizes the box to fit its content, so the dialog grows horizontally as the user types, which looks broken.

Both dialogs should use a proportional fixed width (e.g. `m.width / 2` clamped to a minimum). The `textinput` should also be width-constrained so the text doesn't overflow the box. The `CharLimit = 50` prevents unbounded input but does not constrain rendering width.

Fix: add `.Width(dialogWidth)` to the outer `lipgloss.NewStyle()` in both `renderEditPortfolioDialog` and the inline `creating` dialog, and set `ti.Width = dialogWidth - 6` (accounting for padding and border) in `openEditPortfolioDialog` and `openCreateDialog`.

---

**F3** `internal/ui/portfolio.go:161–164` — `browsing` mode `'a'` handler; `internal/ui/portfolio.go:857` — status bar

The `browsing` mode currently handles `a`/`A` to open the coin picker (add holding). The user reports this is confusing: `n` creates a new portfolio, `a` adds a holding, and both work from the same mode. The UX expectation is that holding management belongs inside the holdings panel (list mode), not from the portfolio navigation level.

Remove the `'a'`, `'A'` case from the `browsing` mode key handler. Remove `• a add` from the `browsing` status bar hint. The `a` shortcut should remain in `listing` mode (already implemented).

Also remove `a add` from the `browsing` status bar string at line 857.

---

**F4** `internal/ui/portfolio.go:192–199` — `browsing` mode `tea.KeyEnter` handler

When a portfolio is selected but has no holdings, pressing Enter is a no-op. The user expects Enter to enter the right panel (holdings area) regardless of whether holdings exist. Currently the code gates on `len(m.holdings) > 0`.

The fix: always transition to `listing{}` on Enter when a portfolio exists, even if holdings are empty. The `listing` mode already handles an empty holdings list gracefully (shows "no holdings — press a to add one"). The `holdingsCursor` clamping handles the empty case. Remove the `len(m.holdings) > 0` guard.

Note: the PRD specifies "Only available when the selected portfolio has at least one holding", but the developer's UX feedback supersedes the PRD here — the panel should always be enterable so users can immediately add holdings after creating a portfolio.
