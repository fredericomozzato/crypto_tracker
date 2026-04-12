---
status: in_review
branch: feat/006-create-portfolio-left-panel
---

# Slice 6 — Create portfolio + left panel

## Context

Slices 1–5 delivered the Markets tab (coin table, auto-refresh, status bar) and the tab bar with an empty Portfolio tab. The Portfolio tab currently renders a single "no portfolios" string with no store dependency and no interaction.

Slice 6 activates the Portfolio tab: a two-panel layout, a left panel listing portfolios with cursor navigation, and a create-portfolio dialog invoked with `n`. The right panel is a placeholder in this slice (holdings are Slice 7).

---

## Scope

Roadmap deliverables for Slice 6:
- Left panel lists portfolios with `▶` cursor; `j`/`k` navigate between them
- `n` opens the create dialog: text input (max 50 chars), `Enter` saves, `Esc` cancels
- After creation: cursor moves to the new portfolio, mode returns to browsing
- TDD: portfolio CRUD in store, dialog state transitions

---

## Data model

**Schema addition** (`internal/db/schema.sql`):

```sql
CREATE TABLE IF NOT EXISTS portfolios (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL,
    created_at INTEGER NOT NULL DEFAULT 0
);
```

No holdings table yet (Slice 7). Foreign keys from holdings to portfolios come in Slice 7.

---

## Files to create / modify

### `internal/db/schema.sql` (modify)

Append the `portfolios` table. The existing `coins` table is unchanged. Schema uses `CREATE TABLE IF NOT EXISTS` so it applies idempotently.

### `internal/store/store.go` (modify)

Add the `Portfolio` struct and extend the `Store` interface:

```go
// Portfolio represents a named collection of holdings.
type Portfolio struct {
    ID        int64
    Name      string
    CreatedAt int64
}

type Store interface {
    // existing methods unchanged
    UpsertCoin(ctx context.Context, c Coin) error
    GetAllCoins(ctx context.Context) ([]Coin, error)
    UpdatePrices(ctx context.Context, prices map[string]float64) error
    Close() error

    // new in Slice 6
    CreatePortfolio(ctx context.Context, name string) (Portfolio, error)
    GetAllPortfolios(ctx context.Context) ([]Portfolio, error)
}
```

### `internal/store/sqlite.go` (modify)

Two new method implementations on `*SQLiteStore`:

**`CreatePortfolio`**:
```go
func (s *SQLiteStore) CreatePortfolio(ctx context.Context, name string) (Portfolio, error) {
    now := time.Now().Unix()
    result, err := s.db.ExecContext(ctx, `
        INSERT INTO portfolios (name, created_at) VALUES (?, ?)
    `, name, now)
    if err != nil {
        return Portfolio{}, fmt.Errorf("creating portfolio %q: %w", name, err)
    }
    id, err := result.LastInsertId()
    if err != nil {
        return Portfolio{}, fmt.Errorf("getting portfolio id: %w", err)
    }
    return Portfolio{ID: id, Name: name, CreatedAt: now}, nil
}
```

**`GetAllPortfolios`**:
```go
func (s *SQLiteStore) GetAllPortfolios(ctx context.Context) ([]Portfolio, error) {
    rows, err := s.db.QueryContext(ctx, `
        SELECT id, name, created_at FROM portfolios ORDER BY created_at ASC
    `)
    if err != nil {
        return nil, fmt.Errorf("querying portfolios: %w", err)
    }
    defer func() { _ = rows.Close() }()

    portfolios := make([]Portfolio, 0)
    for rows.Next() {
        var p Portfolio
        if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt); err != nil {
            return nil, fmt.Errorf("scanning portfolio: %w", err)
        }
        portfolios = append(portfolios, p)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("iterating portfolios: %w", err)
    }
    return portfolios, nil
}
```

### `internal/ui/portfolio.go` (full rewrite)

**Mode discriminated union:**

```go
type portfolioMode interface{ isPortfolioMode() }
type browsing struct{}
type creating struct{ input textinput.Model }

func (browsing) isPortfolioMode() {}
func (creating) isPortfolioMode() {}
```

**Model struct:**

```go
type PortfolioModel struct {
    ctx        context.Context
    store      store.Store
    width      int
    height     int
    portfolios []store.Portfolio
    cursor     int
    mode       portfolioMode
    lastErr    string
}
```

**Constructor:**

```go
func NewPortfolioModel(ctx context.Context, s store.Store) PortfolioModel {
    return PortfolioModel{
        ctx:   ctx,
        store: s,
        mode:  browsing{},
    }
}
```

**Messages:**

```go
// portfoliosLoadedMsg is sent when portfolios are loaded from the store.
// focusID is non-zero when the cursor should be positioned on a specific portfolio.
type portfoliosLoadedMsg struct {
    portfolios []store.Portfolio
    focusID    int64
}
```

**Commands:**

```go
func (m PortfolioModel) cmdLoadPortfolios() tea.Cmd {
    return func() tea.Msg {
        portfolios, err := m.store.GetAllPortfolios(m.ctx)
        if err != nil {
            return errMsg{err: fmt.Errorf("loading portfolios: %w", err)}
        }
        return portfoliosLoadedMsg{portfolios: portfolios}
    }
}

func (m PortfolioModel) cmdCreatePortfolio(name string) tea.Cmd {
    return func() tea.Msg {
        p, err := m.store.CreatePortfolio(m.ctx, name)
        if err != nil {
            return errMsg{err: fmt.Errorf("creating portfolio: %w", err)}
        }
        portfolios, err := m.store.GetAllPortfolios(m.ctx)
        if err != nil {
            return errMsg{err: fmt.Errorf("loading portfolios after create: %w", err)}
        }
        return portfoliosLoadedMsg{portfolios: portfolios, focusID: p.ID}
    }
}
```

**`Init()`:**

```go
func (m PortfolioModel) Init() tea.Cmd {
    return m.cmdLoadPortfolios()
}
```

**`update()` key handling:**

- `tea.WindowSizeMsg` → update width/height
- In `browsing` mode:
  - `j`/`↓` → `moveCursor(+1)`
  - `k`/`↑` → `moveCursor(-1)`
  - `n` → switch to `creating` mode with a new focused `textinput.Model` (max 50 chars, placeholder `e.g. Long Term`)
- In `creating` mode:
  - `Esc` → switch back to `browsing`
  - `Enter` → if `strings.TrimSpace(input.Value()) != ""`, fire `cmdCreatePortfolio`; otherwise no-op
  - All other keys → delegated to `input.Update(msg)`
- `portfoliosLoadedMsg` → update `m.portfolios`, position cursor on `focusID`, clamp cursor, set `m.mode = browsing{}`
- `errMsg` → store in `m.lastErr`; leave mode unchanged

**`InputActive()`:**

```go
func (m PortfolioModel) InputActive() bool {
    _, ok := m.mode.(creating)
    return ok
}
```

**`View()`:**

Two-panel layout with a status bar. Left panel ≈ 30 chars wide; right panel fills the rest. Content height = `m.height - 1` (1 row for status bar).

**Panel borders:** Both panels must be rendered with `lipgloss.NewStyle().Border(lipgloss.NormalBorder())` so the user can always see the two distinct areas, even when holdings are empty. The left panel gets a fixed outer width (including its border); the right panel fills the remaining terminal width. Use `lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)` — no plain space separator between them; the borders provide the visual separation.

Left panel renders:
- If `len(m.portfolios) == 0`: "no portfolios — press n to create one"
- Otherwise: one line per portfolio; the cursor row is rendered with `lipgloss.NewStyle().Reverse(true)` (full-width reverse-video highlight, matching the Markets tab style). No `▶` prefix — the highlight is the sole selection indicator.

Right panel (Slice 6): "no holdings" placeholder, centered vertically and horizontally within the panel.

If mode is `creating`, a centered overlay dialog is rendered on top of the panels:
- Title: "New Portfolio"
- The textinput rendered below title

Status bar (browsing): `j/k portfolios • n new portfolio • q quit`
Status bar (creating): `Enter to create • Esc to cancel`

If `m.lastErr != ""`, append ` • error: <msg>` to the status bar.

### `internal/ui/app.go` (modify)

Two changes:

1. `NewAppModel` passes `ctx` and `s` to `NewPortfolioModel`:

```go
func NewAppModel(ctx context.Context, s store.Store, c api.CoinGeckoClient) AppModel {
    return AppModel{
        activeTab: tabMarkets,
        markets:   NewMarketsModel(ctx, s, c),
        portfolio: NewPortfolioModel(ctx, s),
    }
}
```

2. `Init()` batches both children:

```go
func (m AppModel) Init() tea.Cmd {
    return tea.Batch(m.markets.Init(), m.portfolio.Init())
}
```

### `internal/ui/testhelpers_test.go` (modify)

Extend `StubStore` with portfolio fields and methods:

```go
// add field
portfolios []store.Portfolio

func (s *StubStore) CreatePortfolio(ctx context.Context, name string) (store.Portfolio, error) {
    if s.err != nil {
        return store.Portfolio{}, s.err
    }
    p := store.Portfolio{
        ID:        int64(len(s.portfolios) + 1),
        Name:      name,
        CreatedAt: time.Now().Unix(),
    }
    s.portfolios = append(s.portfolios, p)
    return p, nil
}

func (s *StubStore) GetAllPortfolios(ctx context.Context) ([]store.Portfolio, error) {
    if s.err != nil {
        return nil, s.err
    }
    return s.portfolios, nil
}
```

---

## Tests

### `internal/store/sqlite_test.go` — new test cases

**`TestCreatePortfolio`**
Creates one portfolio with name `"Long Term"`, calls `GetAllPortfolios`, verifies `len == 1`, `Name == "Long Term"`, `ID > 0`.

**`TestCreatePortfolioSetsCreatedAt`**
Creates a portfolio, reads back, verifies `CreatedAt != 0`.

**`TestGetAllPortfoliosEmpty`**
Empty DB: `GetAllPortfolios` returns a non-nil empty slice with `len == 0`.

**`TestGetAllPortfoliosMultiple`**
Creates 3 portfolios (`"A"`, `"B"`, `"C"`), verifies `GetAllPortfolios` returns all 3 ordered by `created_at ASC` (creation order preserved).

**`TestCreatePortfolioReturnsInsertedID`**
Creates two portfolios; verifies the second has a higher `ID` than the first.

### `internal/ui/portfolio_test.go` — new/replaced test cases

The existing 6 tests cover the Slice 5 skeleton and will be replaced as the model changes significantly.

**`TestNewPortfolioModel`**
Verifies zero dimensions, cursor=0, mode is `browsing`, no portfolios.

**`TestPortfolioInputActiveFalseWhenBrowsing`**
`InputActive()` returns `false` on a freshly-created model.

**`TestPortfolioInputActiveTrueWhenCreating`**
After sending `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}`, `InputActive()` returns `true`.

**`TestPortfolioNKeyOpensCreateDialog`**
After `n` keypress, mode is `creating`.

**`TestPortfolioCreateDialogEscCancels`**
In `creating` mode, `Esc` returns mode to `browsing`.

**`TestPortfolioCreateDialogEnterWithEmptyIsNoOp`**
In `creating` mode with empty input, `Enter` returns nil cmd and stays in `creating` mode.

**`TestPortfolioCreateDialogEnterWithTextReturnsCmd`**
In `creating` mode with `"My Portfolio"` typed into the input, `Enter` returns a non-nil cmd.

**`TestPortfolioJKNavigation`**
Load a model with 3 portfolios via `portfoliosLoadedMsg`. Send `j` twice → cursor at 2. Send `k` → cursor at 1.

**`TestPortfolioCursorClampsAtTop`**
Cursor at 0, send `k` → cursor stays at 0.

**`TestPortfolioCursorClampsAtBottom`**
Cursor at last portfolio, send `j` → cursor stays at last.

**`TestPortfoliosLoadedMsgPopulatesModel`**
Send `portfoliosLoadedMsg{portfolios: threePortfolios()}`, verify `len(m.portfolios) == 3`.

**`TestPortfoliosLoadedMsgPositionsCursorOnFocusID`**
Three portfolios with IDs 1, 2, 3. Send `portfoliosLoadedMsg{focusID: 3}`, verify `cursor == 2`.

**`TestPortfoliosLoadedMsgSwitchesToBrowsing`**
Model in `creating` mode receives `portfoliosLoadedMsg`, mode becomes `browsing`.

**`TestPortfolioViewShowsPortfolioNames`**
Model with two portfolios loaded. `View()` contains both names.

**`TestPortfolioViewShowsCursorIndicator`**
Model with portfolios loaded, cursor at 0. The selected portfolio name appears in the output (highlight is applied via `lipgloss.Reverse`; the test verifies the name is present and that the non-selected row does not carry the same styling as the selected row). No `▶` glyph is expected.

**`TestPortfolioViewShowsEmptyStateWhenNoPortfolios`**
No portfolios loaded. `View()` contains "no portfolios".

**`TestPortfolioHandlesWindowSizeMsg`**
`WindowSizeMsg{Width: 120, Height: 39}` → `m.width == 120`, `m.height == 39`.

---

## Implementation order

1. **Schema** — append `portfolios` table to `internal/db/schema.sql`
2. **Store interface** — add `Portfolio` struct + `CreatePortfolio`/`GetAllPortfolios` to `store.go`
3. **Store tests (RED)** — write 5 new failing tests in `sqlite_test.go`
4. **Store implementation (GREEN)** — implement methods in `sqlite.go`; all store tests pass
5. **Testhelpers update** — add portfolio fields/methods to `StubStore` in `testhelpers_test.go`
6. **Portfolio UI tests (RED)** — write new failing tests in `portfolio_test.go`
7. **Portfolio UI (GREEN)** — rewrite `portfolio.go` with mode union, `update()`, `View()`, `Init()`
8. **App wiring** — update `app.go`: new `NewPortfolioModel` call + batch `Init()`
9. **Verify** — `make check`; all tests pass, race detector clean

---

## Verification

```bash
make check
# expect: gofumpt clean, golangci-lint clean, all tests pass, govulncheck clean

go test -race -v ./internal/store/... -run "Portfolio"
# expect: 5 new portfolio store tests pass

go test -race -v ./internal/ui/... -run "Portfolio"
# expect: ~17 new portfolio UI tests pass

go run ./cmd/crypto-tracker
# visual check:
# - Portfolio tab shows two-panel layout
# - 'n' opens text input dialog
# - Type a name, Enter → portfolio appears in left panel with ▶ cursor
# - Esc in dialog → returns to browsing without creating
# - j/k navigate the list
```
