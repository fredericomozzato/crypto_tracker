---
status: done
branch: feat/003-markets-table-scrolling-formatting
---

# Slice 3 — Markets table: 100 coins, scrolling, formatting

## Context

Slices 1 and 2 delivered the skeleton app and the full data pipeline for a single coin. `AppModel` fetches 1 coin from `/coins/markets`, upserts it to SQLite, reads it back, and renders it with raw `fmt.Sprintf`. The `r` key refreshes prices via `/simple/price`. The `Store` interface, `SQLiteStore`, `CoinGeckoClient` interface, and `db` package are all in place.

This slice scales the pipeline to 100 coins, adds cursor navigation with viewport scrolling, introduces the `format` package for consistent number formatting, and renders a proper column table. The "load from DB on subsequent launches" logic lands here too, making the app feel fast after the first run.

## Scope

From the roadmap:
- Fetch top 100 coins on first launch; load from DB on subsequent launches
- Scrollable table: `j`/`k`/`↓`/`↑`/`g`/`G`, cursor highlighting
- `internal/format/format.go` — `FmtPrice`, `FmtChange` with proper thresholds
- TDD: format functions, cursor movement/clamping logic

**Rate limiting constraint:** Only batch requests are permitted. The initial seed uses `/coins/markets?per_page=100` — a single call that returns all data. Price refreshes use `/simple/price?ids=<all ids>` — already a single batched call. Individual per-coin requests must never be added.

## Files to create / modify

### 1. `internal/format/format.go` *(new)*

Purpose: shared formatting helpers used by the Markets table (and future Portfolio table). No lipgloss dependency — colors are applied in the View layer.

```go
package format

// FmtPrice formats a USD price:
//   - v >= 1: "$X,XXX.XX" (2 dp, comma thousands separator)
//   - v < 1:  "$0.XXXXXX" (6 dp)
func FmtPrice(v float64) string

// FmtChange formats a 24 h percentage change as "+X.XX%" or "-X.XX%".
func FmtChange(v float64) string
```

`FmtPrice` implementation notes:
- Format integer part with `fmt.Sprintf` then insert commas: walk backwards inserting a `,` every 3 digits.
- Concatenate `$`, integer part with commas, `.`, and decimal part.
- No external dependency — pure stdlib.

`FmtChange` implementation notes:
- `v >= 0` → prefix with `+`; negative values get the `-` from `fmt.Sprintf("%.2f")` naturally.

### 2. `internal/format/format_test.go` *(new)*

Each test calls the function and compares the result to the expected string.

| Test | Input | Expected output |
|------|-------|-----------------|
| `TestFmtPriceAboveOne` | `67234.56` | `"$67,234.56"` |
| `TestFmtPriceMillions` | `1234567.89` | `"$1,234,567.89"` |
| `TestFmtPriceExactlyOne` | `1.0` | `"$1.00"` |
| `TestFmtPriceBelowOne` | `0.00012345` | `"$0.000123"` (6 dp) |
| `TestFmtPriceSmallAboveOne` | `1.50` | `"$1.50"` |
| `TestFmtChangePositive` | `2.34` | `"+2.34%"` |
| `TestFmtChangeNegative` | `-1.23` | `"-1.23%"` |
| `TestFmtChangeZero` | `0.0` | `"+0.00%"` |

### 3. `internal/ui/app.go` *(modify)*

**New fields on `AppModel`:**

```go
type AppModel struct {
    width      int
    height     int
    ctx        context.Context
    store      store.Store
    client     api.CoinGeckoClient
    coins      []store.Coin
    lastErr    string
    refreshing bool
    cursor     int  // index of selected row in m.coins
    offset     int  // index of first visible row (viewport scroll)
}
```

**`Init` — load-or-fetch logic:**

```go
func (m AppModel) Init() tea.Cmd {
    return func() tea.Msg {
        existing, err := m.store.GetAllCoins(m.ctx)
        if err != nil {
            return errMsg{err: fmt.Errorf("loading coins: %w", err)}
        }
        if len(existing) > 0 {
            // Subsequent launch: serve from DB, no network request
            return coinsLoadedMsg{coins: existing}
        }
        // First launch: fetch 100 coins, upsert all, return
        fetched, err := m.client.FetchMarkets(m.ctx, 100)
        if err != nil {
            return errMsg{err: err}
        }
        for _, c := range fetched {
            if err := m.store.UpsertCoin(m.ctx, c); err != nil {
                return errMsg{err: fmt.Errorf("upserting coin %s: %w", c.ApiID, err)}
            }
        }
        stored, err := m.store.GetAllCoins(m.ctx)
        if err != nil {
            return errMsg{err: fmt.Errorf("loading coins after seed: %w", err)}
        }
        return coinsLoadedMsg{coins: stored}
    }
}
```

**`Update` — new key handling inside the existing `switch msg.Type` block:**

```go
case tea.KeyDown:
    m.moveCursor(+1)
case tea.KeyUp:
    m.moveCursor(-1)
case tea.KeyRunes:
    for _, r := range msg.Runes {
        switch r {
        case 'q': return m, tea.Quit
        case 'j': m.moveCursor(+1)
        case 'k': m.moveCursor(-1)
        case 'g': m.cursor = 0; m.adjustViewport()
        case 'G':
            if len(m.coins) > 0 {
                m.cursor = len(m.coins) - 1
                m.adjustViewport()
            }
        case 'r':
            if !m.refreshing && len(m.coins) > 0 {
                m.refreshing = true
                return m, m.cmdRefresh()
            }
        }
    }
```

**New helpers on `AppModel`:**

```go
// moveCursor moves the cursor by delta and adjusts the viewport.
func (m *AppModel) moveCursor(delta int) {
    m.cursor += delta
    if m.cursor < 0 { m.cursor = 0 }
    if m.cursor >= len(m.coins) { m.cursor = len(m.coins) - 1 }
    m.adjustViewport()
}

// adjustViewport updates m.offset so the cursor row stays visible.
func (m *AppModel) adjustViewport() {
    h := m.tableHeight()
    if m.cursor < m.offset {
        m.offset = m.cursor
    }
    if m.cursor >= m.offset+h {
        m.offset = m.cursor - h + 1
    }
    maxOff := len(m.coins) - h
    if maxOff < 0 { maxOff = 0 }
    if m.offset > maxOff { m.offset = maxOff }
    if m.offset < 0 { m.offset = 0 }
}

// tableHeight returns the number of rows available for coin data.
// Reserves 1 row for column headers and 1 row for the hint line.
func (m AppModel) tableHeight() int {
    h := m.height - 2
    if h < 1 { return 1 }
    return h
}
```

**`View` — table rendering:**

When coins are loaded, render:
1. A header row with fixed-width columns: `#`, `Name`, `Ticker`, `Price (USD)`, `24h`
2. `tableHeight()` visible coin rows starting at `m.offset`; the selected row gets a lipgloss reverse/highlight style
3. The 24h change column rendered in green (positive) or red (negative) using lipgloss `Foreground`
4. A single hint line at the bottom: `"j/k navigate • g/G top/bottom • r refresh • q quit"`

Column widths (fixed):

| Column | Width | Alignment |
|--------|-------|-----------|
| `#` | 4 | right |
| `Name` | 22 | left, truncate at 20 chars + `…` |
| `Ticker` | 8 | left |
| `Price (USD)` | 14 | right |
| `24h` | 9 | right |

Total: ~57 chars — well within the 100-column minimum.

Name truncation helper: `truncate(s string, maxLen int) string` — if `len(s) > maxLen`, return `s[:maxLen-1] + "…"`.

When no coins and no error: render `"loading..."` centered (existing behavior).
When error: render error string (existing behavior).

### 4. `internal/ui/app_test.go` *(modify)*

**Update `StubAPI`** to track `FetchMarkets` calls:

```go
type StubAPI struct {
    coins             []store.Coin
    prices            map[string]float64
    err               error
    fetchMarketsCalls []int // records the limit arg each time FetchMarkets is called
}

func (a *StubAPI) FetchMarkets(ctx context.Context, limit int) ([]store.Coin, error) {
    a.fetchMarketsCalls = append(a.fetchMarketsCalls, limit)
    if a.err != nil { return nil, a.err }
    return a.coins, nil
}
```

**New tests:**

| Test | Setup | Action | Assert |
|------|-------|--------|--------|
| `TestCursorMovesDownOnJ` | 3 coins loaded, cursor=0 | send `j` key | cursor==1 |
| `TestCursorMovesUpOnK` | 3 coins loaded, cursor=1 | send `k` key | cursor==0 |
| `TestCursorClampsAtBottom` | 3 coins, cursor=2 | send `j` key | cursor still 2 |
| `TestCursorClampsAtTop` | 3 coins, cursor=0 | send `k` key | cursor still 0 |
| `TestCursorJumpsToTopOnG` | 3 coins, cursor=2 | send `g` key | cursor==0 |
| `TestCursorJumpsToBottomOnCapG` | 3 coins, cursor=0 | send `G` key | cursor==2 |
| `TestCursorMovesDownOnDownArrow` | 3 coins, cursor=0 | send `tea.KeyDown` | cursor==1 |
| `TestCursorMovesUpOnUpArrow` | 3 coins, cursor=1 | send `tea.KeyUp` | cursor==0 |
| `TestInitFetchesHundredCoinsOnFirstLaunch` | StubStore empty, StubAPI with coins | call Init and execute cmd | `fetchMarketsCalls[0]==100`, result is `coinsLoadedMsg` |
| `TestInitLoadsFromDBOnSubsequentLaunch` | StubStore with 3 coins pre-populated | call Init and execute cmd | `fetchMarketsCalls` is empty, result is `coinsLoadedMsg` with the 3 DB coins |
| `TestViewRendersColumnHeaders` | 3 coins loaded, width=120, height=40 | call View | output contains "Name", "Ticker", "Price", "24h" |
| `TestViewRendersHintLine` | 3 coins loaded, width=120, height=40 | call View | output contains "j/k" |

**Existing tests to update:**
- `TestCoinsLoadedMsg` — price is now rendered through `FmtPrice`; update the assertion from `"67000"` to `"$67,000.00"`. Also assert column headers are present.

## Implementation order

1. Write `internal/format/format_test.go` (all tests, all red)
2. Write `internal/format/format.go` — make format tests green
3. Add `cursor`/`offset` fields and helper methods (`moveCursor`, `adjustViewport`, `tableHeight`) to `AppModel` in `internal/ui/app.go` — do not change `Init` or `View` yet
4. Add cursor-movement tests to `internal/ui/app_test.go` — should be green against the new helpers
5. Update `Init` in `app.go` to check DB first / fetch 100 on first launch
6. Add `TestInitFetchesHundredCoinsOnFirstLaunch` and `TestInitLoadsFromDBOnSubsequentLaunch` — should be green
7. Replace `View` in `app.go` with the full table renderer (imports `format`)
8. Add/update view tests (`TestViewRendersColumnHeaders`, `TestViewRendersHintLine`, update `TestCoinsLoadedMsg`)
9. Run `make check` — all tests pass, no lint errors

## Verification

```bash
make check                        # fmt + lint + test + vuln — must all pass
make build                        # produces ./crypto-tracker binary
go run ./cmd/crypto-tracker       # first run: fetches 100 coins, renders scrollable table
                                  # j/k/g/G: cursor moves, viewport scrolls
                                  # r: refreshes prices
                                  # q: quits cleanly
go run ./cmd/crypto-tracker       # second run: loads from DB instantly, no API call on start
```
