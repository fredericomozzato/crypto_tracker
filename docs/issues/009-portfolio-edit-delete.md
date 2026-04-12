---
status: done
branch: feat/009-portfolio-edit-delete
---

# Slice 9 — Portfolio Management (Edit and Delete)

## Context

Slices 1–8 are complete. The portfolio tab supports browsing, creating, adding/editing/deleting holdings, and two-panel layout with focus highlighting. The `PortfolioModel` uses a discriminated union `portfolioMode` with 7 states: `browsing`, `creating`, `addCoin`, `addAmount`, `listing`, `editingAmount`, `deleting`. Slice 9 adds the ability to **rename** and **delete** portfolios from the left panel, plus a `UNIQUE` constraint on portfolio names.

Since we are in development mode, no production data migration is needed — we can delete the existing DB and let the schema re-create from scratch.

## Scope

From roadmap:
- `e` in browsing mode → open edit dialog for selected portfolio (pre-populated with current name)
- `Esc` cancels edit → returns to browsing
- `Enter` saves: empty/whitespace-only = no-op return to browsing; same name = no-op return to browsing; duplicate name = inline error message; unique name = save and return to browsing
- `X` in browsing mode → delete confirmation dialog showing portfolio name
- `Enter` confirms delete → portfolio is deleted, returns to browsing
- `Esc` cancels delete → returns to browsing
- After delete: portfolios reload, cursor repositioned (clamped), holdings reload for new selection (or empty state if no portfolios remain)
- TDD: name uniqueness, edit/delete state transitions

## Data Model

### Schema change — `internal/db/schema.sql`

Add `UNIQUE` constraint to `portfolios.name`:

```sql
CREATE TABLE IF NOT EXISTS portfolios (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL UNIQUE,
    created_at INTEGER NOT NULL DEFAULT 0
);
```

No migration code needed in `db.go`. Since we are in development, delete the existing DB file and let `CREATE TABLE IF NOT EXISTS` rebuild it with the new constraint.

### Store interface additions — `internal/store/store.go`

```go
RenamePortfolio(ctx context.Context, id int64, name string) error
DeletePortfolio(ctx context.Context, id int64) error
```

### `RenamePortfolio` — `internal/store/sqlite.go`

```go
func (s *SQLiteStore) RenamePortfolio(ctx context.Context, id int64, name string) error {
    result, err := s.db.ExecContext(ctx,
        `UPDATE portfolios SET name = ? WHERE id = ?`, name, id,
    )
    if err != nil {
        return fmt.Errorf("renaming portfolio: %w", err)
    }
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("renaming portfolio %d: %w", id, err)
    }
    if rows == 0 {
        return fmt.Errorf("renaming portfolio %d: not found", id)
    }
    return nil
}
```

The `UNIQUE` constraint on `name` will cause a duplicate-name error from SQLite. The UI pre-validates to give a friendly inline error message; the DB constraint is the safety net.

### `DeletePortfolio` — `internal/store/sqlite.go`

```go
func (s *SQLiteStore) DeletePortfolio(ctx context.Context, id int64) error {
    _, err := s.db.ExecContext(ctx, `DELETE FROM portfolios WHERE id = ?`, id)
    if err != nil {
        return fmt.Errorf("deleting portfolio %d: %w", id, err)
    }
    return nil
}
```

Cascade (`ON DELETE CASCADE` on `holdings.portfolio_id`) automatically deletes all holdings for the portfolio.

## New mode types — `internal/ui/portfolio.go`

```go
type editingPortfolio struct {
    portfolio store.Portfolio
    input     textinput.Model
    errMsg    string
}

type deletingPortfolio struct {
    portfolio store.Portfolio
}

func (editingPortfolio) isPortfolioMode()  {}
func (deletingPortfolio) isPortfolioMode()   {}
```

## New message type — `internal/ui/portfolio.go`

```go
type portfolioDeletedMsg struct {
    portfolios []store.Portfolio
}
```

## Key handler changes — `internal/ui/portfolio.go`

### `browsing` mode additions

```go
case 'e', 'E':
    if len(m.portfolios) > 0 {
        m = m.openEditPortfolioDialog()
        return m, nil
    }
case 'X', 'x':
    if len(m.portfolios) > 0 {
        m.mode = deletingPortfolio{portfolio: m.portfolios[m.cursor]}
        return m, nil
    }
```

### `editingPortfolio` mode handler

```go
case editingPortfolio:
    switch msg.Type {
    case tea.KeyEsc:
        m.mode = browsing{}
        return m, nil
    case tea.KeyEnter:
        name := strings.TrimSpace(mode.input.Value())
        if name == "" || name == mode.portfolio.Name {
            m.mode = browsing{}
            return m, nil
        }
        for _, p := range m.portfolios {
            if p.Name == name && p.ID != mode.portfolio.ID {
                mode.errMsg = "name already exists"
                m.mode = mode
                return m, nil
            }
        }
        return m, m.cmdRenamePortfolio(mode.portfolio.ID, name)
    default:
        newInput, cmd := mode.input.Update(msg)
        mode.input = newInput
        m.mode = mode
        return m, cmd
    }
```

### `deletingPortfolio` mode handler

```go
case deletingPortfolio:
    switch msg.Type {
    case tea.KeyEsc:
        m.mode = browsing{}
        return m, nil
    case tea.KeyEnter:
        return m, m.cmdDeletePortfolio(mode.portfolio.ID)
    }
    // All other keys ignored
```

### New message handler — `portfolioDeletedMsg`

```go
case portfolioDeletedMsg:
    m.portfolios = msg.portfolios
    m.mode = browsing{}
    m.scrollOffset = 0
    m.holdingsCursor = 0
    if len(m.portfolios) == 0 {
        m.cursor = 0
        m.holdings = nil
        m.lastErr = ""
        return m, nil
    }
    if m.cursor >= len(m.portfolios) {
        m.cursor = len(m.portfolios) - 1
    }
    if m.cursor < 0 {
        m.cursor = 0
    }
    m.lastErr = ""
    return m, m.cmdLoadHoldings(m.portfolios[m.cursor].ID)
```

The `portfoliosLoadedMsg` handler is reused for rename (with `focusID` set to the renamed portfolio's ID).

### New command functions

```go
func (m PortfolioModel) cmdRenamePortfolio(portfolioID int64, name string) tea.Cmd {
    return func() tea.Msg {
        if err := m.store.RenamePortfolio(m.ctx, portfolioID, name); err != nil {
            return errMsg{err: fmt.Errorf("renaming portfolio: %w", err)}
        }
        portfolios, err := m.store.GetAllPortfolios(m.ctx)
        if err != nil {
            return errMsg{err: fmt.Errorf("loading portfolios after rename: %w", err)}
        }
        return portfoliosLoadedMsg{portfolios: portfolios, focusID: portfolioID}
    }
}

func (m PortfolioModel) cmdDeletePortfolio(portfolioID int64) tea.Cmd {
    return func() tea.Msg {
        if err := m.store.DeletePortfolio(m.ctx, portfolioID); err != nil {
            return errMsg{err: fmt.Errorf("deleting portfolio: %w", err)}
        }
        portfolios, err := m.store.GetAllPortfolios(m.ctx)
        if err != nil {
            return errMsg{err: fmt.Errorf("loading portfolios after delete: %w", err)}
        }
        return portfolioDeletedMsg{portfolios: portfolios}
    }
}
```

### `openEditPortfolioDialog` helper

```go
func (m PortfolioModel) openEditPortfolioDialog() PortfolioModel {
    ti := textinput.New()
    ti.Placeholder = "e.g. Long Term"
    ti.CharLimit = 50
    ti.Focus()
    ti.SetValue(m.portfolios[m.cursor].Name)
    m.mode = editingPortfolio{
        portfolio: m.portfolios[m.cursor],
        input:     ti,
    }
    return m
}
```

### `InputActive()` update

```go
case creating, addCoin, addAmount, editingAmount, editingPortfolio:
    return true
```

### `View()` update — new dialog rendering

```go
case editingPortfolio:
    dialog := m.renderEditPortfolioDialog(mode)
    content := lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, dialog)
    return content + "\n" + m.renderStatusBar()

case deletingPortfolio:
    dialog := m.renderDeletePortfolioDialog(mode)
    content := lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, dialog)
    return content + "\n" + m.renderStatusBar()
```

### Dialog renderers

```go
func (m PortfolioModel) renderEditPortfolioDialog(mode editingPortfolio) string {
    var b strings.Builder
    b.WriteString("Rename Portfolio\n\n")
    b.WriteString(mode.input.View() + "\n")
    if mode.errMsg != "" {
        b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(mode.errMsg))
    }
    return lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        Padding(1, 2).
        Render(b.String())
}

func (m PortfolioModel) renderDeletePortfolioDialog(mode deletingPortfolio) string {
    var b strings.Builder
    b.WriteString("Delete Portfolio\n\n")
    b.WriteString(mode.portfolio.Name + "\n\n")
    b.WriteString("Press Enter to confirm, or Esc to cancel.")
    return lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        Padding(1, 2).
        Render(b.String())
}
```

### Status bar update

```go
case browsing:
    content = "j/k portfolios • Enter list • e edit • X delete • PgUp/PgDn scroll • a add • n new • q quit"
case editingPortfolio:
    content = "Enter to save • Esc to cancel"
case deletingPortfolio:
    content = "Enter to delete • Esc to cancel"
```

### Border highlighting

Both `editingPortfolio` and `deletingPortfolio` are launched from browsing mode (left panel focus). In the `View()` focus determination, both should show the left panel as focused:

```go
case browsing, creating, addCoin, addAmount, editingPortfolio, deletingPortfolio:
    leftFocused = true
```

## Tests

### `internal/store/sqlite_test.go`

- **`TestRenamePortfolio`** — rename a portfolio, verify name updated via `GetAllPortfolios`
- **`TestRenamePortfolioDuplicateName`** — create two portfolios, try to rename one to the other's name, expect error
- **`TestRenamePortfolioNotFound`** — rename a nonexistent ID, expect error
- **`TestDeletePortfolio`** — create a portfolio, delete it, verify it's gone
- **`TestDeletePortfolioCascadeHoldings`** — create portfolio + coin + holding, delete portfolio, verify holding was cascade-deleted
- **`TestDeletePortfolioNotFound`** — delete nonexistent ID, no error (idempotent)
- **`TestCreatePortfolioDuplicateName`** — verify that creating two portfolios with the same name fails (UNIQUE constraint)

### `internal/ui/portfolio_test.go`

- **`TestBrowsingEKeyOpensEditDialog`** — press `e` in browsing mode with portfolios → transitions to `editingPortfolio`
- **`TestBrowsingEKeyNoPortfoliosIsNoOp`** — press `e` with no portfolios → no-op
- **`TestEditingPortfolioEscReturnsToBrowsing`** — Esc from edit dialog returns to browsing
- **`TestEditingPortfolioEnterWithEmptyNameReturnsToBrowsing`** — empty name after trim → return to browsing
- **`TestEditingPortfolioEnterWithSameNameReturnsToBrowsing`** — unchanged name → return to browsing
- **`TestEditingPortfolioEnterWithDuplicateNameShowsError`** — type name of another portfolio, Enter, `errMsg` is set
- **`TestEditingPortfolioEnterWithValidNameReturnsCmd`** — type a valid new name, Enter, cmd is non-nil
- **`TestEditingPortfolioInputActive`** — `editingPortfolio` mode → `InputActive()` returns true
- **`TestEditingPortfolioPrePopulatedName`** — the text input contains the current portfolio name
- **`TestBrowsingXKeyOpensDeleteDialog`** — press `X` in browsing mode → transitions to `deletingPortfolio`
- **`TestBrowsingXKeyNoPortfoliosIsNoOp`** — press `X` with no portfolios → no-op
- **`TestDeletingPortfolioEscReturnsToBrowsing`** — Esc from delete dialog returns to browsing
- **`TestDeletingPortfolioEnterReturnsCmd`** — Enter in delete dialog returns non-nil command
- **`TestDeletingPortfolioOtherKeysIgnored`** — other keys (j, k, a, n, 1) are silently ignored
- **`TestDeletingPortfolioInputActive`** — `deletingPortfolio` mode → `InputActive()` returns false
- **`TestPortfolioDeletedMsgUpdatesPortfolios`** — `portfolioDeletedMsg` replaces `m.portfolios`
- **`TestPortfolioDeletedMsgClampsCursor`** — if cursor >= len after delete, it's clamped
- **`TestPortfolioDeletedMsgResetsScrollOffset`** — scroll offset is reset to 0 after delete
- **`TestPortfolioDeletedMsgToBrowsingWhenEmpty`** — if all portfolios deleted, mode is browsing with empty holdings
- **`TestPortfolioDeletedMsgLoadsHoldingsForNewSelection`** — after delete with remaining portfolios, cmd loads holdings
- **`TestEditPortfolioDialogShowsCurrentName`** — `View()` in `editingPortfolio` mode contains the portfolio name
- **`TestDeletePortfolioDialogShowsName`** — `View()` in `deletingPortfolio` mode contains the portfolio name

### `internal/ui/testhelpers_test.go`

Add stub methods to `StubStore`:

```go
func (s *StubStore) RenamePortfolio(ctx context.Context, id int64, name string) error {
    if s.err != nil {
        return s.err
    }
    for i, p := range s.portfolios {
        if p.ID == id {
            s.portfolios[i].Name = name
            return nil
        }
    }
    return nil
}

func (s *StubStore) DeletePortfolio(ctx context.Context, id int64) error {
    for i, p := range s.portfolios {
        if p.ID == id {
            s.portfolios = append(s.portfolios[:i], s.portfolios[i+1:]...)
            return nil
        }
    }
    return nil
}
```

### `internal/db/db_test.go`

Add test for the UNIQUE constraint:

- **`TestUniquePortfolioName`** — verify that creating two portfolios with the same name fails

## Implementation Order

1. Update `schema.sql` — add `UNIQUE` to `portfolios.name`
2. Add `RenamePortfolio` and `DeletePortfolio` to `Store` interface in `store.go`
3. Implement `RenamePortfolio` and `DeletePortfolio` in `sqlite.go`
4. Write store tests in `sqlite_test.go` (TDD: write tests first, then verify implementation passes)
5. Add `editingPortfolio` and `deletingPortfolio` mode types, `portfolioDeletedMsg`, and `isPortfolioMode()` implementations to `portfolio.go`
6. Add `RenamePortfolio` and `DeletePortfolio` stub methods to `testhelpers_test.go`
7. Write UI tests for state transitions in `portfolio_test.go` (TDD: all tests red at this point)
8. Implement `browsing` mode key handlers (`e`, `X`)
9. Implement `editingPortfolio` and `deletingPortfolio` update handlers
10. Implement `portfolioDeletedMsg` handler
11. Implement `cmdRenamePortfolio` and `cmdDeletePortfolio`
12. Implement `openEditPortfolioDialog` helper
13. Update `InputActive()`, `View()`, `renderStatusBar()`
14. Write view tests (`TestEditPortfolioDialogShowsCurrentName`, `TestDeletePortfolioDialogShowsName`)
15. Delete existing development DB file and run `make check` — all tests pass, lint clean, no race conditions

## Verification

```bash
make check
go test -race -run TestRename -v ./internal/store/
go test -race -run TestDeletePortfolio -v ./internal/store/
go test -race -run TestEditingPortfolio -v ./internal/ui/
go test -race -run TestDeletingPortfolio -v ./internal/ui/
go test -race -run TestBrowsingE -v ./internal/ui/
go test -race -run TestBrowsingX -v ./internal/ui/
go test -race -run TestPortfolioDeleted -v ./internal/ui/
```