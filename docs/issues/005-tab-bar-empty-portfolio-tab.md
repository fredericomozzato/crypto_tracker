---
status: in_progress
branch: feat/005-tab-bar-empty-portfolio-tab
---

# Slice 5 — Tab bar + empty Portfolio tab

## Context

Slices 1–4 built a fully functional Markets tab: 100 coins from CoinGecko, scrollable table with cursor navigation, auto-refresh every 60 s via a 5 s ticker, error propagation through a typed `errMsg`, and a status bar with `Synced`/`Stale`/`Refreshing`/`error:` states. All of this currently lives in a single `internal/ui/app.go` (`AppModel`).

Slice 5 introduces the shell needed to host two tabs. That means:
- Turning `AppModel` into a true root/router model (tab bar, tab switching, global quit, input suppression)
- Extracting all markets logic into a new `MarketsModel` in `markets.go`
- Creating a minimal `PortfolioModel` in `portfolio.go` (empty state only — full content comes in Slices 6–8)
- Moving test helpers and markets tests out of `app_test.go` and into dedicated files

## Scope

From the roadmap:
- Two tabs rendered at the top: `[ Markets ]  [ Portfolio ]`
- `Tab` / `Shift+Tab` / `1` / `2` switch tabs
- Portfolio tab shows empty state: `"no portfolios — press n to create one"`
- Tab switching suppressed when `InputActive()` returns true (for future dialog use)
- **TDD:** tab switching logic, input suppression

## Data model

No schema changes. No new DB tables.

---

## Files to create

### `internal/ui/markets.go`

All market-specific state, commands, messages, and rendering extracted from the current `app.go`.

**Key types and signatures:**

```go
// MarketsModel manages the Markets tab: coin list, auto-refresh, cursor, status bar.
type MarketsModel struct {
    width         int
    height        int
    ctx           context.Context
    store         store.Store
    client        api.CoinGeckoClient
    coins         []store.Coin
    lastErr       string
    refreshing    bool
    lastRefreshed time.Time
    cursor        int
    offset        int
}

// Messages (moved from app.go):
type coinsLoadedMsg   struct{ coins []store.Coin }
type errMsg           struct{ err error }
type pricesUpdatedMsg struct{ coins []store.Coin }
type tickMsg          time.Time

const staleThreshold = 5 * time.Minute

func NewMarketsModel(ctx context.Context, s store.Store, c api.CoinGeckoClient) MarketsModel

// Init returns the batched load + tick commands.
func (m MarketsModel) Init() tea.Cmd

// update handles all markets messages. Returns typed MarketsModel, not tea.Model.
// Does NOT handle 'q' or Ctrl+C — those belong to AppModel.
func (m MarketsModel) update(msg tea.Msg) (MarketsModel, tea.Cmd)

// InputActive always returns false — Markets has no text inputs.
func (m MarketsModel) InputActive() bool

// View renders the coin table + status bar. Assumes height set via WindowSizeMsg.
func (m MarketsModel) View() string

// Internal helpers (moved from app.go):
func (m MarketsModel) cmdLoad() tea.Cmd
func (m MarketsModel) cmdRefresh() tea.Cmd
func (m MarketsModel) statusRight() string
func (m MarketsModel) renderStatusBar() string
func (m *MarketsModel) moveCursor(delta int)
func (m *MarketsModel) adjustViewport()
func (m MarketsModel) tableHeight() int
```

Key points:
- `update` handles: `tea.WindowSizeMsg`, `tea.KeyMsg` (j/k/g/G/r/arrows), `tickMsg`, `coinsLoadedMsg`, `pricesUpdatedMsg`, `errMsg`
- `update` does **not** handle `q` or `Ctrl+C` — AppModel owns quit
- `View` does **not** check terminal size — AppModel guards that
- `tableHeight()` uses `m.height - 2` (header row + status bar row), same as today

### `internal/ui/portfolio.go`

```go
// PortfolioModel is the Portfolio tab. Slice 5: empty state only.
type PortfolioModel struct {
    width  int
    height int
}

func NewPortfolioModel() PortfolioModel

// update handles tea.WindowSizeMsg; ignores all other messages.
func (m PortfolioModel) update(msg tea.Msg) (PortfolioModel, tea.Cmd)

// InputActive always returns false in this slice (no dialogs yet).
func (m PortfolioModel) InputActive() bool

// View renders the empty-state message.
// → "no portfolios — press n to create one"
func (m PortfolioModel) View() string
```

### `internal/ui/testhelpers_test.go`

Shared test fixtures used by both `app_test.go` and `markets_test.go`. Extracted from the current `app_test.go` to avoid duplicate declarations:

```go
// StubStore, StubAPI, makeCoins(), threeCoins(), setupMarketsModel()
```

`setupMarketsModel` replaces the current `setupCursorModel`, operating on `MarketsModel` directly.

### `internal/ui/markets_test.go`

All markets-specific tests from `app_test.go`, adapted to call `MarketsModel` directly:

| Test | What it verifies |
|------|-----------------|
| `TestMarketsInit` | `Init()` returns a non-nil batched cmd |
| `TestMarketsCoinsLoadedMsg` | `coinsLoadedMsg` populates coins, sets `lastRefreshed`, renders coin names |
| `TestMarketsErrMsg` | `errMsg` sets `lastErr`, clears `refreshing`, view contains error text |
| `TestMarketsViewRendersLoading` | Empty coins → view contains `"loading"` |
| `TestMarketsViewRendersColumnHeaders` | View contains `#`, `Name`, `Ticker`, `Price (USD)`, `24h` |
| `TestMarketsViewRendersHintLine` | View contains `"j/k"` hint |
| `TestMarketsRefreshKey` | `r` with loaded coins → `refreshing=true`, non-nil cmd |
| `TestMarketsRefreshKeyIgnoredWhenAlreadyRefreshing` | `r` while `refreshing=true` → nil cmd |
| `TestMarketsRefreshKeyIgnoredWhenNoCoins` | `r` with no coins → nil cmd |
| `TestMarketsPricesUpdatedMsg` | Updates coins, clears `refreshing`, sets `lastRefreshed` |
| `TestMarketsViewShowsRefreshHint` | View contains `"r refresh"` |
| `TestMarketsCursorMovesDownOnJ` | `j` → cursor +1 |
| `TestMarketsCursorMovesUpOnK` | `k` → cursor -1 |
| `TestMarketsCursorClampsAtBottom` | `j` at last item → stays |
| `TestMarketsCursorClampsAtTop` | `k` at first item → stays |
| `TestMarketsCursorJumpsToTopOnG` | `g` → cursor = 0 |
| `TestMarketsCursorJumpsToBottomOnCapG` | `G` → cursor = last |
| `TestMarketsCursorMovesDownOnDownArrow` | `KeyDown` → cursor +1 |
| `TestMarketsCursorMovesUpOnUpArrow` | `KeyUp` → cursor -1 |
| `TestMarketsMoveCursorNoPanicOnEmptyCoins` | j/k with empty slice → no panic, cursor stays 0 |
| `TestMarketsCursorClampedAfterCoinsLoaded` | cursor beyond slice length → clamped |
| `TestMarketsTickMsgAlwaysReissuesTicker` | `tickMsg` always returns a cmd |
| `TestMarketsTickMsgBelow60sDoesNotRefresh` | 30s elapsed → `refreshing` stays false |
| `TestMarketsTickMsgAbove60sFiresRefresh` | 61s elapsed → `refreshing=true`, cmd returned |
| `TestMarketsTickMsgWhenAlreadyRefreshing` | Already refreshing → no second refresh |
| `TestMarketsTickMsgWhenNoCoins` | No coins → no refresh |
| `TestMarketsCoinsLoadedSetsLastRefreshed` | `lastRefreshed` set after load |
| `TestMarketsPricesUpdatedSetsLastRefreshed` | `lastRefreshed` set after price update |
| `TestMarketsStatusBarShowsLoading` | `statusRight()` returns `"loading..."` before first load |
| `TestMarketsStatusBarShowsRefreshing` | `statusRight()` returns `"Refreshing"` |
| `TestMarketsStatusBarShowsError` | `statusRight()` returns `"error: …"` |
| `TestMarketsStatusBarShowsSynced` | `statusRight()` returns `"Synced"` |
| `TestMarketsStatusBarShowsStale` | `statusRight()` returns `"Stale"` after 5+ minutes |
| `TestMarketsTableRendersWhileError` | Error state still shows coin rows |
| `TestMarketsStatusBarHasHintsOnLeft` | View contains `"j/k navigate"` |
| `TestMarketsInitFetchesHundredCoinsOnFirstLaunch` | Empty DB → `FetchMarkets(100)` called |
| `TestMarketsInitLoadsFromDBOnSubsequentLaunch` | 100 coins in DB → no API call |
| `TestMarketsInitRefetchesWhenDBPartiallySeeded` | <100 coins in DB → `FetchMarkets` called |
| `TestMarketsIgnoresOtherKeys` | Keys a/b/c/x/z/1/2/' ' → nil cmd (tab switching is AppModel's job) |
| `TestMarketsInputActiveFalse` | `InputActive()` always returns false |

### `internal/ui/portfolio_test.go`

| Test | What it verifies |
|------|-----------------|
| `TestNewPortfolioModel` | `NewPortfolioModel()` returns zero-value model |
| `TestPortfolioViewShowsEmptyState` | `View()` contains `"no portfolios"` |
| `TestPortfolioViewShowsCreateHint` | `View()` contains `"press n to create"` |
| `TestPortfolioInputActiveFalse` | `InputActive()` returns false |
| `TestPortfolioHandlesWindowSizeMsg` | `update(WindowSizeMsg{120, 39})` → `width=120, height=39` |
| `TestPortfolioUpdateIgnoresOtherMessages` | Arbitrary key msg → nil cmd, model unchanged |

---

## Files to modify

### `internal/ui/app.go`

Becomes a lean root/router model:

```go
type tab int

const (
    tabMarkets   tab = iota
    tabPortfolio
)

const tabCount = 2

// AppModel is the root Bubble Tea model. Owns tab bar, tab routing, global quit.
type AppModel struct {
    width     int
    height    int
    activeTab tab
    markets   MarketsModel
    portfolio PortfolioModel
}

func NewAppModel(ctx context.Context, s store.Store, c api.CoinGeckoClient) AppModel {
    return AppModel{
        activeTab: tabMarkets,
        markets:   NewMarketsModel(ctx, s, c),
        portfolio: NewPortfolioModel(),
    }
}

// Init delegates to the Markets model's Init (only tab that does I/O on startup).
func (m AppModel) Init() tea.Cmd

// Update handles: WindowSizeMsg (propagated to both children with height-1),
// tab switching keys (Tab/Shift+Tab/1/2), global quit (q/Ctrl+C),
// and delegates all other messages to the active child model.
// Tab switching is suppressed when the active child's InputActive() returns true.
// Ctrl+C always quits regardless of InputActive().
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)

// View renders: terminal-too-small guard, then tab bar + active child view.
func (m AppModel) View() string

// renderTabBar renders "[ Markets ]  [ Portfolio ]" with active tab highlighted.
func (m AppModel) renderTabBar() string

// activeInputActive returns whether the currently active child has a text input focused.
func (m AppModel) activeInputActive() bool
```

**`Update` logic:**

```go
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // WindowSizeMsg: store full size, forward adjusted size to both children
    if ws, ok := msg.(tea.WindowSizeMsg); ok {
        m.width = ws.Width
        m.height = ws.Height
        childMsg := tea.WindowSizeMsg{Width: ws.Width, Height: ws.Height - 1}
        var cmd1, cmd2 tea.Cmd
        m.markets, cmd1 = m.markets.update(childMsg)
        m.portfolio, cmd2 = m.portfolio.update(childMsg)
        return m, tea.Batch(cmd1, cmd2)
    }

    if key, ok := msg.(tea.KeyMsg); ok {
        // Ctrl+C always quits
        if key.Type == tea.KeyCtrlC {
            return m, tea.Quit
        }
        // Global keys only active when no text input is focused
        if !m.activeInputActive() {
            switch key.Type {
            case tea.KeyTab:
                m.activeTab = tab((int(m.activeTab) + 1) % tabCount)
                return m, nil
            case tea.KeyShiftTab:
                m.activeTab = tab((int(m.activeTab) - 1 + tabCount) % tabCount)
                return m, nil
            case tea.KeyRunes:
                switch string(key.Runes) {
                case "q":
                    return m, tea.Quit
                case "1":
                    m.activeTab = tabMarkets
                    return m, nil
                case "2":
                    m.activeTab = tabPortfolio
                    return m, nil
                }
            }
        }
    }

    // Delegate all other messages to the active child
    switch m.activeTab {
    case tabMarkets:
        var cmd tea.Cmd
        m.markets, cmd = m.markets.update(msg)
        return m, cmd
    case tabPortfolio:
        var cmd tea.Cmd
        m.portfolio, cmd = m.portfolio.update(msg)
        return m, cmd
    }
    return m, nil
}
```

**`View` logic:**

```go
func (m AppModel) View() string {
    if m.width < 100 || m.height < 30 {
        return "Terminal too small — resize to at least 100×30"
    }
    tabBar := m.renderTabBar()
    switch m.activeTab {
    case tabMarkets:
        return tabBar + "\n" + m.markets.View()
    case tabPortfolio:
        return tabBar + "\n" + m.portfolio.View()
    }
    return tabBar
}
```

### `internal/ui/app_test.go`

Stripped down to only `AppModel`-level concerns. All markets tests removed (moved to `markets_test.go`). New tests:

| Test | What it verifies |
|------|-----------------|
| `TestNewAppModel` | Creates model without panic, view is non-empty |
| `TestAppModelDefaultTabIsMarkets` | `activeTab == tabMarkets` after construction |
| `TestAppModelViewContainsTabBar` | View contains `"Markets"` and `"Portfolio"` labels |
| `TestQuitOnQ` | `q` → `tea.QuitMsg` |
| `TestQuitOnCtrlC` | `Ctrl+C` → `tea.QuitMsg` |
| `TestTabKeyAdvancesToPortfolio` | `Tab` from Markets → `activeTab == tabPortfolio` |
| `TestTabKeyWrapsToMarkets` | `Tab` from Portfolio → `activeTab == tabMarkets` |
| `TestShiftTabGoesBack` | `Shift+Tab` from Portfolio → `activeTab == tabMarkets` |
| `TestShiftTabWrapsToPortfolio` | `Shift+Tab` from Markets → `activeTab == tabPortfolio` |
| `TestOneKeySelectsMarkets` | `"1"` → `activeTab == tabMarkets` |
| `TestTwoKeySelectsPortfolio` | `"2"` → `activeTab == tabPortfolio` |
| `TestActiveInputActiveFalseByDefault` | `activeInputActive()` returns false when both children idle |
| `TestCtrlCQuitsFromPortfolioTab` | `Ctrl+C` quits from portfolio tab |
| `TestQuitFromPortfolioTab` | `q` quits from portfolio tab |
| `TestWindowSizeMsgSetsRootDimensions` | `WindowSizeMsg` sets `m.width` and `m.height` on `AppModel` |
| `TestWindowSizeMsgPropagatesAdjustedHeightToChildren` | After `WindowSizeMsg{120, 40}`, child markets model has `height == 39` |
| `TestInitReturnsBatchedCmd` | `Init()` returns non-nil cmd |

Note: Direct input-suppression integration testing (Tab blocked during text input) is deferred to Slice 6 when `PortfolioModel` gains real text inputs that make `InputActive()` return true.

---

## Implementation order

1. **Create `internal/ui/testhelpers_test.go`** — move `StubStore`, `StubAPI`, `makeCoins`, `threeCoins` from `app_test.go`; add `setupMarketsModel`
2. **Write `internal/ui/portfolio_test.go`** — all portfolio tests (red)
3. **Create `internal/ui/portfolio.go`** — make portfolio tests green
4. **Write `internal/ui/markets_test.go`** — all markets tests adapted from `app_test.go` (red)
5. **Create `internal/ui/markets.go`** — extract markets logic from `app.go`; make markets tests green
6. **Update `internal/ui/app_test.go`** — remove duplicate tests, add new tab switching tests (red)
7. **Refactor `internal/ui/app.go`** — root model with tab bar, switching, delegation; make app tests green
8. **Run `make check`** — all tests pass, lint clean

## Verification

```bash
make check
# Expected: gofumpt clean, golangci-lint passes, all tests pass with race detector

go run ./cmd/crypto-tracker
# Expected:
# - Tab bar visible at top with "[ Markets ]  [ Portfolio ]"
# - Active tab visually highlighted
# - Tab / Shift+Tab / 1 / 2 switch between tabs
# - Portfolio tab shows: "no portfolios — press n to create one"
# - Markets tab behaves identically to before (table, auto-refresh, status bar)
# - q and Ctrl+C quit from either tab
```
