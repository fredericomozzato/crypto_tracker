---
status: in_review
branch: feat/007-add-holding-coin-picker-amount-input
---

## Context

Slices 1–6 deliver a working two-tab app with a Markets table (auto-refreshing 100
coins from CoinGecko) and a Portfolio tab with a left panel listing portfolios
(created/navigated). Holdings exist only as a placeholder right panel. This slice
completes the "add holding" flow end-to-end: coin picker dialog → amount input →
upsert → holdings table rendered in the right panel.

---

## Scope

From roadmap slice 7:

- `a` opens coin picker dialog (searchable, filterable list of all coins)
- Select coin → amount input → upsert holding
- Right panel shows holdings table: Coin, Ticker, Amount, Price, Value, 24h, %
- Holdings ordered by value descending, portfolio total in header
- TDD: holding upsert (including update-on-conflict), filter logic, computed values

**Duplicate prevention:** the coin picker only shows coins not yet held in the
current portfolio. The DB enforces `UNIQUE(portfolio_id, coin_id)` as a safety
net; the UI enforces it by filtering the picker list. If all coins are already
held, the picker does not open and a status-bar error is shown instead.

---

## Data model

**New `holdings` table** — added to `internal/db/schema.sql`:

```sql
CREATE TABLE IF NOT EXISTS holdings (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    portfolio_id INTEGER NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    coin_id      INTEGER NOT NULL REFERENCES coins(id) ON DELETE CASCADE,
    amount       REAL    NOT NULL,
    UNIQUE(portfolio_id, coin_id)
);
```

The `UNIQUE(portfolio_id, coin_id)` constraint enforces uniqueness at the DB
level. `ON DELETE CASCADE` on both foreign keys ensures holdings are automatically
removed when a portfolio or coin is deleted (used by future slices).

---

## Files to create / modify

### `internal/db/schema.sql`

Append the `CREATE TABLE IF NOT EXISTS holdings` DDL at the end. No other change.

---

### `internal/store/store.go`

Add two new types:

```go
// Holding represents a coin position in a portfolio (raw row, no joins).
type Holding struct {
    ID          int64
    PortfolioID int64
    CoinID      int64
    Amount      float64
}

// HoldingRow is a holding joined with coin data and with computed portfolio metrics.
type HoldingRow struct {
    ID          int64
    PortfolioID int64
    CoinID      int64
    ApiID       string
    Name        string
    Ticker      string
    Amount      float64
    Rate        float64
    PriceChange float64
    Value       float64 // Amount * Rate, computed in SQL
    Proportion  float64 // Value / portfolio_total * 100, computed via window function
}
```

Extend `Store` interface:

```go
UpsertHolding(ctx context.Context, portfolioID, coinID int64, amount float64) error
DeleteHolding(ctx context.Context, id int64) error
GetHoldingsForPortfolio(ctx context.Context, portfolioID int64) ([]HoldingRow, error)
```

---

### `internal/store/sqlite.go`

**`UpsertHolding`** — pure upsert, no prior read needed:

```go
func (s *SQLiteStore) UpsertHolding(ctx context.Context, portfolioID, coinID int64, amount float64) error {
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO holdings (portfolio_id, coin_id, amount)
        VALUES (?, ?, ?)
        ON CONFLICT(portfolio_id, coin_id) DO UPDATE SET
            amount = excluded.amount
    `, portfolioID, coinID, amount)
    if err != nil {
        return fmt.Errorf("upserting holding (portfolio=%d coin=%d): %w", portfolioID, coinID, err)
    }
    return nil
}
```

**`DeleteHolding`** — by primary key:

```go
func (s *SQLiteStore) DeleteHolding(ctx context.Context, id int64) error {
    _, err := s.db.ExecContext(ctx, `DELETE FROM holdings WHERE id = ?`, id)
    if err != nil {
        return fmt.Errorf("deleting holding %d: %w", id, err)
    }
    return nil
}
```

**`GetHoldingsForPortfolio`** — JOIN + window function for proportion:

```go
func (s *SQLiteStore) GetHoldingsForPortfolio(ctx context.Context, portfolioID int64) ([]HoldingRow, error) {
    rows, err := s.db.QueryContext(ctx, `
        SELECT
            h.id,
            h.portfolio_id,
            h.coin_id,
            c.api_id,
            c.name,
            c.ticker,
            h.amount,
            c.rate,
            c.price_change,
            h.amount * c.rate AS value,
            CASE
                WHEN SUM(h.amount * c.rate) OVER () = 0 THEN 0
                ELSE (h.amount * c.rate) / SUM(h.amount * c.rate) OVER () * 100
            END AS proportion
        FROM holdings h
        JOIN coins c ON c.id = h.coin_id
        WHERE h.portfolio_id = ?
        ORDER BY value DESC
    `, portfolioID)
    // scan into []HoldingRow; return non-nil empty slice when no rows
}
```

Returns `[]HoldingRow{}` (non-nil empty slice) when the portfolio has no holdings.

---

### `internal/format/format.go`

Add `FmtMoney`:

```go
// FmtMoney formats a holding value as "$X,XXX.XX" (always 2 dp, thousands separator).
func FmtMoney(v float64) string {
    parts := strings.SplitN(fmt.Sprintf("%.2f", v), ".", 2)
    return "$" + addCommas(parts[0]) + "." + parts[1]
}
```

---

### `internal/ui/portfolio.go`

**New mode types** appended to the discriminated union:

```go
type (
    browsing  struct{}
    creating  struct{ input textinput.Model }
    addCoin   struct {
        filter   textinput.Model
        allCoins []store.Coin // already filtered — held coins removed
        filtered []store.Coin // subset matching current filter query
        cursor   int
    }
    addAmount struct {
        coin     store.Coin
        input    textinput.Model
        errMsg   string
        coinMode addCoin // preserved so Esc returns to coin picker with state intact
    }
)

func (addCoin) isPortfolioMode()   {}
func (addAmount) isPortfolioMode() {}
```

**Updated `PortfolioModel` struct** — add `holdings` field:

```go
type PortfolioModel struct {
    ctx        context.Context
    store      store.Store
    width      int
    height     int
    portfolios []store.Portfolio
    cursor     int
    holdings   []store.HoldingRow
    mode       portfolioMode
    lastErr    string
}
```

**New messages:**

```go
type coinPickerReadyMsg struct{ coins []store.Coin }
type holdingsLoadedMsg  struct{ holdings []store.HoldingRow }
type holdingsSavedMsg   struct{ holdings []store.HoldingRow }
```

**Updated `InputActive()`:**

```go
func (m PortfolioModel) InputActive() bool {
    switch m.mode.(type) {
    case creating, addCoin, addAmount:
        return true
    }
    return false
}
```

**`update` changes:**

In `browsing` mode:
- `'a'` key: guard — only fire cmd when `len(m.portfolios) > 0`; return `m, m.cmdOpenCoinPicker()`
- `'j'`/`'k'`/`↓`/`↑`: after moving cursor, also return `m.cmdLoadHoldings(currentPortfolioID)`

Handle `coinPickerReadyMsg`:
- Build `map[int64]bool` of already-held `CoinID`s from `m.holdings`
- Strip held coins from `msg.coins` → `available`
- If `len(available) == 0`: set `m.lastErr = "all coins already in portfolio"`, return without entering picker
- Otherwise: create fresh `addCoin` mode (filter textinput focused, `allCoins = available`, `filtered = available`, `cursor = 0`)

In `addCoin` mode:
- `Esc` → `browsing{}`
- `Enter` → if `len(mode.filtered) > 0`, transition to `addAmount` (amount textinput focused, `coinMode = mode`)
- `j`/`↓` → `mode.cursor = min(mode.cursor+1, len(mode.filtered)-1)`
- `k`/`↑` → `mode.cursor = max(mode.cursor-1, 0)`
- Any other key → delegate to `mode.filter.Update(msg)`, recompute `mode.filtered = filterCoins(mode.allCoins, mode.filter.Value())`, clamp `mode.cursor`

In `addAmount` mode:
- `Esc` → restore `m.mode = mode.coinMode`
- `Enter` → parse input as float64; if parse error or value ≤ 0: set `mode.errMsg`, stay in `addAmount`; if valid: return `m, m.cmdUpsertHolding(currentPortfolioID, mode.coin.ID, amount)`
- Any other key → delegate to `mode.input.Update(msg)`

Handle `holdingsLoadedMsg`: `m.holdings = msg.holdings`

Handle `holdingsSavedMsg`: `m.holdings = msg.holdings`, switch to `browsing{}`

Handle `portfoliosLoadedMsg` (existing): after updating cursor, also return `m.cmdLoadHoldings(currentPortfolioID)` as a batched cmd

**Filter helper** (package-level, pure function — easy to test directly):

```go
func filterCoins(coins []store.Coin, query string) []store.Coin {
    if query == "" {
        return coins
    }
    q := strings.ToLower(query)
    result := make([]store.Coin, 0)
    for _, c := range coins {
        if strings.Contains(strings.ToLower(c.Name), q) ||
            strings.Contains(strings.ToLower(c.Ticker), q) ||
            strings.Contains(strings.ToLower(c.ApiID), q) {
            result = append(result, c)
        }
    }
    return result
}
```

**New commands:**

```go
// cmdOpenCoinPicker loads all coins from the store. The Update handler strips
// already-held coins before entering addCoin mode.
func (m PortfolioModel) cmdOpenCoinPicker() tea.Cmd

// cmdLoadHoldings fetches holdings for the given portfolio ID.
func (m PortfolioModel) cmdLoadHoldings(portfolioID int64) tea.Cmd

// cmdUpsertHolding saves the holding and reloads the holdings list.
func (m PortfolioModel) cmdUpsertHolding(portfolioID, coinID int64, amount float64) tea.Cmd
```

**Right panel rendering** (`renderRightPanel`):

- Header: selected portfolio name + total value (`FmtMoney(sum of HoldingRow.Value)`)
- Empty state: `"no holdings — press a to add one"`
- Holdings table rows: Coin | Ticker | Amount | Price | Value | 24h | %
  - Amount: `fmt.Sprintf("%.4f", h.Amount)`
  - Price: `format.FmtPrice(h.Rate)`
  - Value: `format.FmtMoney(h.Value)`
  - 24h: `format.FmtChange(h.PriceChange)` (green/red via lipgloss)
  - %: `fmt.Sprintf("%.1f%%", h.Proportion)`

**Dialog overlays** (rendered as centered lipgloss overlays, same pattern as `creating`):

- `addCoin`: title `"Select Coin"`, filter input at top, scrollable coin list below
- `addAmount`: selected coin name + ticker, amount input, optional inline error line below input

**Status bar additions:**

```go
case addCoin{}:
    content = "j/k navigate • type to filter • Enter select • Esc cancel"
case addAmount{}:
    content = "Enter to confirm • Esc back to coin selection"
```

---

### `internal/ui/testhelpers_test.go`

Add `holdingRows []store.HoldingRow` field to `StubStore`. Implement the three new
interface methods:

```go
func (s *StubStore) UpsertHolding(ctx context.Context, portfolioID, coinID int64, amount float64) error
func (s *StubStore) DeleteHolding(ctx context.Context, id int64) error
func (s *StubStore) GetHoldingsForPortfolio(ctx context.Context, portfolioID int64) ([]store.HoldingRow, error)
```

`GetHoldingsForPortfolio` returns `s.holdingRows` directly (ignores `portfolioID`).
This lets tests pre-seed the result without needing realistic joins. Tests that need
realistic computed values set up `holdingRows` manually.

---

## Tests

### `internal/store/sqlite_test.go`

**`TestUpsertHoldingInsertsNew`** — insert a holding for a known coin and portfolio;
`GetHoldingsForPortfolio` returns it with correct `Amount`, `CoinID`, `Name`, `Ticker`.

**`TestUpsertHoldingUpdatesOnConflict`** — upsert the same `(portfolio_id, coin_id)`
twice with different amounts; only one row exists with the latest amount.

**`TestGetHoldingsForPortfolioEmpty`** — portfolio with no holdings returns non-nil
empty slice.

**`TestGetHoldingsForPortfolioJoinsCoinData`** — `Name`, `Ticker`, `Rate`,
`PriceChange` are populated correctly from the coins table.

**`TestGetHoldingsComputedValue`** — amount=2, coin rate=500 → `Value=1000.0`.

**`TestGetHoldingsOrderedByValueDesc`** — three holdings with different values; slice
is sorted largest-first.

**`TestGetHoldingsProportion`** — two holdings with values 1000 and 3000 →
proportions 25.0 and 75.0 respectively.

**`TestGetHoldingsProportionZeroWhenNoValue`** — holdings with zero-rate coins →
proportion 0, no division-by-zero.

**`TestDeleteHolding`** — upsert a holding, delete it by ID,
`GetHoldingsForPortfolio` returns empty slice.

**`TestDeleteHoldingNonExistentIsNoOp`** — deleting an ID that doesn't exist does not
error.

### `internal/format/format_test.go`

**`TestFmtMoneyZero`** — `FmtMoney(0)` → `"$0.00"`.

**`TestFmtMoneySmall`** — `FmtMoney(0.5)` → `"$0.50"`.

**`TestFmtMoneyThousands`** — `FmtMoney(12345.678)` → `"$12,345.68"`.

### `internal/ui/portfolio_test.go`

**`TestPortfolioAKeyWhenNoPortfoliosIsNoOp`** — pressing `a` with no portfolios
returns nil cmd.

**`TestPortfolioAKeyOpensCoinPicker`** — with at least one portfolio, pressing `a`
in browsing mode returns a non-nil cmd.

**`TestCoinPickerReadyMsgEntersAddCoinMode`** — `coinPickerReadyMsg{coins: threeCoins()}`
(none held) → mode is `addCoin`, `InputActive()` is true.

**`TestCoinPickerReadyMsgWithNoCoinsShowsError`** — `coinPickerReadyMsg{coins: nil}`
→ stays `browsing{}`, `m.lastErr` non-empty.

**`TestCoinPickerFiltersOutAlreadyHeldCoins`** — model with one `HoldingRow` for
coin ID 1 (BTC); `coinPickerReadyMsg` with BTC + ETH → `addCoin.allCoins` contains
only ETH.

**`TestCoinPickerAllHeldShowsError`** — every coin in `coinPickerReadyMsg` is already
held → stays `browsing{}`, `m.lastErr` non-empty, `InputActive()` false.

**`TestCoinPickerEscReturnsToBrowsing`** — `Esc` in `addCoin` mode → `browsing{}`.

**`TestCoinPickerJKNavigatesCursor`** — 3 filtered coins, two `j` presses → cursor=2.

**`TestCoinPickerCursorClampsAtTop`** — `k` at cursor=0 → cursor stays 0.

**`TestCoinPickerCursorClampsAtBottom`** — `j` at last item → cursor stays at last.

**`TestCoinPickerTypingFilters`** — typing a character reduces `mode.filtered` to
matching coins only.

**`TestCoinPickerCursorClampedAfterFilter`** — cursor at 2 before a filter yielding
1 result → cursor clamped to 0.

**`TestCoinPickerEnterTransitionsToAddAmount`** — `Enter` in `addCoin` mode with ≥1
filtered coin → `addAmount` mode.

**`TestAddAmountEscReturnsToCoinPicker`** — `Esc` in `addAmount` → restores `addCoin`
mode with filter and cursor intact.

**`TestAddAmountEnterWithEmptyIsNoOp`** — `Enter` on empty input → stays `addAmount`,
nil cmd.

**`TestAddAmountEnterWithNonNumericSetsInlineError`** — "abc" + `Enter` → stays
`addAmount`, `mode.errMsg` non-empty.

**`TestAddAmountEnterWithZeroOrNegativeSetsInlineError`** — "0" and "-1" each set
`mode.errMsg`.

**`TestAddAmountEnterWithValidAmountReturnsCmd`** — "1.5" + `Enter` → non-nil cmd.

**`TestHoldingsSavedMsgReturnsToBrowsing`** — `holdingsSavedMsg` → mode is
`browsing{}`.

**`TestHoldingsSavedMsgUpdatesHoldings`** — `holdingsSavedMsg{holdings: rows}` →
`m.holdings` populated.

**`TestHoldingsLoadedMsgUpdatesHoldings`** — `holdingsLoadedMsg{holdings: rows}` →
`m.holdings` populated.

**`TestFilterCoinsEmptyQuery`** — empty query returns full list unchanged.

**`TestFilterCoinsByName`** — case-insensitive substring match on `Name`.

**`TestFilterCoinsByTicker`** — case-insensitive match on `Ticker`.

**`TestFilterCoinsByApiID`** — case-insensitive match on `ApiID`.

**`TestFilterCoinsNoMatch`** — no match returns non-nil empty slice.

**`TestPortfolioViewShowsHoldingsTable`** — `m.holdings` populated → `View()` contains
coin name and formatted value.

**`TestPortfolioViewShowsNoHoldingsMessage`** — empty `m.holdings` with a portfolio
selected → right panel contains "no holdings".

**`TestPortfolioInputActiveForAddCoinMode`** — mode is `addCoin{}` → `InputActive()`
true.

**`TestPortfolioInputActiveForAddAmountMode`** — mode is `addAmount{}` →
`InputActive()` true.

---

## Implementation order

1. `schema.sql` — add `holdings` table DDL
2. `store/store.go` — add `Holding`, `HoldingRow` types; extend `Store` interface
3. `store/sqlite_test.go` — write 10 new holding tests (red)
4. `store/sqlite.go` — implement `UpsertHolding`, `DeleteHolding`, `GetHoldingsForPortfolio` (green)
5. `format/format.go` + `format/format_test.go` — add `FmtMoney` and its tests
6. `ui/testhelpers_test.go` — extend `StubStore` with holdings methods + `holdingRows` field
7. `ui/portfolio_test.go` — write all new UI tests (red)
8. `ui/portfolio.go` — implement `filterCoins`, new modes, mode transitions, commands, right panel, status bar, `InputActive()` (green)
9. `make check` — fmt + lint + test + vuln must all pass

---

## Verification

```bash
make check
# Expected: all tests pass, no lint errors, no vulnerabilities

go test -v ./internal/store/... | grep -E "^(=== RUN|--- PASS|--- FAIL|FAIL|ok)"
# Expected: all existing + 10 new store tests PASS

go test -v ./internal/ui/... | grep -E "^(=== RUN|--- PASS|--- FAIL|FAIL|ok)"
# Expected: all existing + ~25 new UI tests PASS

go test -v ./internal/format/... | grep -E "^(=== RUN|--- PASS|--- FAIL|FAIL|ok)"
# Expected: all existing + 3 new format tests PASS

go run ./cmd/crypto-tracker
# Manual smoke:
# 1. Switch to Portfolio tab, create a portfolio
# 2. Press 'a' → coin picker opens with full coin list
# 3. Type partial name → list filters in real time
# 4. Select coin with Enter → amount input appears
# 5. Enter invalid amount → inline error shown, stays on dialog
# 6. Enter valid amount → holdings table appears in right panel
# 7. Press 'a' again → already-held coin absent from picker list
# 8. If all coins held → picker does not open, error in status bar
```
