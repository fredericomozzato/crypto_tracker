---
status: in_progress
branch: feat/004-auto-refresh-status-bar
---

## Slice 4 — Auto-refresh + status bar

### Context

Slices 1–3 built the skeleton, full data pipeline, and markets table with 100 scrollable coins. The current `AppModel` already has:
- `refreshing bool` and `lastErr string` fields
- `cmdRefresh()` to batch-update prices via `/simple/price`
- Manual `r` key handler that guards against double-refresh
- `errMsg` / `pricesUpdatedMsg` message types

What's missing for Slice 4:
1. **Auto-refresh ticker** — 5 s tick → check if 60 s elapsed → fire `cmdRefresh`
2. **`lastRefreshed time.Time`** field to track when data was last loaded
3. **Proper two-sided status bar** replacing the current combined hint line:
   - Left: keyboard hints (`j/k navigate • g/G top/bottom • r refresh • q quit`)
   - Right: sync status (`synced Xs ago` / `refreshing...` / `error: <message>` / `loading...`)
4. **Table always renders** when coins are present, even if `lastErr != ""` — errors go in the status bar only, not as full-screen replacements

---

### Scope

From roadmap bullets:
- 5 s ticker → checks if 60 s elapsed → fires `cmdRefresh` via `/simple/price`
- Manual refresh with `r` (no-op if already refreshing) — already implemented, just wired to status bar
- Status bar: `synced Xs ago` / `refreshing...` / `error: <message>` / `loading...`
- Error propagation via typed `errMsg`, non-fatal — keeps table usable with stale data
- **TDD:** tick/refresh state transitions, error display logic

---

### Files to modify

#### `internal/ui/app.go`

**Purpose:** Add ticker, `lastRefreshed`, and replace the simple hint line with a proper two-sided status bar.

**Changes:**

New field on `AppModel`:
```go
type AppModel struct {
    // existing fields unchanged …
    lastRefreshed time.Time
}
```

New message type:
```go
// tickMsg fires every 5 s from cmdTick.
type tickMsg time.Time
```

New constructor function (package-level, not method — no `AppModel` context needed):
```go
func cmdTick() tea.Cmd {
    return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}
```

Updated `Init()` — batch the data-load command with the first tick:
```go
func (m AppModel) Init() tea.Cmd {
    return tea.Batch(m.cmdLoad(), cmdTick())
}
```

`cmdLoad()` — extract existing anonymous init function to a named method so `Init()` stays clean:
```go
func (m AppModel) cmdLoad() tea.Cmd {
    return func() tea.Msg { /* existing Init body */ }
}
```

Updated `Update()` — add `tickMsg` case and update `lastRefreshed` on success:

```go
case tickMsg:
    cmds := []tea.Cmd{cmdTick()} // always re-arm the ticker
    if !m.refreshing && len(m.coins) > 0 && time.Since(m.lastRefreshed) >= 60*time.Second {
        m.refreshing = true
        cmds = append(cmds, m.cmdRefresh())
    }
    return m, tea.Batch(cmds...)

case coinsLoadedMsg:
    m.coins = msg.coins
    m.lastErr = ""
    m.lastRefreshed = time.Now()
    // existing cursor clamping …

case pricesUpdatedMsg:
    m.coins = msg.coins
    m.refreshing = false
    m.lastErr = ""
    m.lastRefreshed = time.Now()
```

New `statusRight()` helper (priority: refreshing > error > loading > synced):
```go
func (m AppModel) statusRight() string {
    if m.refreshing {
        return "refreshing..."
    }
    if m.lastErr != "" {
        return "error: " + m.lastErr
    }
    if m.lastRefreshed.IsZero() {
        return "loading..."
    }
    elapsed := time.Since(m.lastRefreshed)
    switch {
    case elapsed < time.Minute:
        return fmt.Sprintf("synced %ds ago", int(elapsed.Seconds()))
    case elapsed < time.Hour:
        return fmt.Sprintf("synced %dm ago", int(elapsed.Minutes()))
    default:
        return fmt.Sprintf("synced %dh ago", int(elapsed.Hours()))
    }
}
```

New `renderStatusBar()` helper — left/right layout using `lipgloss`:
```go
func (m AppModel) renderStatusBar() string {
    leftContent := "j/k navigate • g/G top/bottom • r refresh • q quit"
    rightContent := m.statusRight()

    grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
    errStyle  := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))

    var rightStyled string
    if m.lastErr != "" && !m.refreshing {
        rightStyled = errStyle.Render(rightContent)
    } else {
        rightStyled = grayStyle.Render(rightContent)
    }

    leftStyled := grayStyle.Render(leftContent)
    padding := m.width - lipgloss.Width(leftContent) - lipgloss.Width(rightContent)
    if padding < 1 {
        padding = 1
    }
    return leftStyled + strings.Repeat(" ", padding) + rightStyled
}
```

Updated `View()` — remove the early-return error block; always render the table when terminal is big enough; use `renderStatusBar()`:
```go
func (m AppModel) View() string {
    if m.width < 100 || m.height < 30 {
        return "Terminal too small — resize to at least 100×30"
    }

    // … existing column/header rendering …

    // When no coins yet, table body is empty rows — status bar shows "loading..."
    // existing viewport logic unchanged

    // Replace hint line with status bar
    return strings.Join(lines, "\n") + "\n" + m.renderStatusBar()
}
```

Imports to add: `"time"`

---

#### `internal/ui/app_test.go`

**Purpose:** Cover all new tick/refresh state transitions and status bar display logic.

**Test cases to add:**

| Test name | What it verifies |
|---|---|
| `TestTickMsgAlwaysReissuesTicker` | On any `tickMsg`, returned `cmd` is non-nil (ticker is re-armed) |
| `TestTickMsgBelow60sDoesNotRefresh` | `tickMsg` when `lastRefreshed` is 30 s ago → `refreshing` stays false |
| `TestTickMsgAbove60sFiresRefresh` | `tickMsg` when `lastRefreshed` is 61 s ago → `refreshing = true`, cmd is non-nil |
| `TestTickMsgWhenAlreadyRefreshing` | `tickMsg` when `refreshing = true` → `refreshing` unchanged, only ticker re-armed |
| `TestTickMsgWhenNoCoins` | `tickMsg` when `coins` is empty → no refresh even if 60 s elapsed |
| `TestCoinsLoadedSetsLastRefreshed` | `coinsLoadedMsg` sets `lastRefreshed` to a non-zero time |
| `TestPricesUpdatedSetsLastRefreshed` | `pricesUpdatedMsg` sets `lastRefreshed` to a non-zero time |
| `TestStatusBarShowsLoading` | View with zero coins and no error → status bar right contains `loading...` |
| `TestStatusBarShowsRefreshing` | View with `refreshing = true` → status bar right contains `refreshing...` |
| `TestStatusBarShowsError` | View with `lastErr = "some error"` → status bar right contains `error: some error` |
| `TestStatusBarShowsSyncedAgo` | View after `coinsLoadedMsg` → status bar right contains `synced` and `ago` |
| `TestTableRendersWhileError` | View with coins loaded + `lastErr != ""` → view still contains coin names AND error text |
| `TestInitReturnsBatchedCmd` | `Init()` returns a non-nil cmd (batch of load + tick) |
| `TestStatusBarHasHintsOnLeft` | View with coins loaded → status bar contains `j/k navigate` |

**Existing tests to note:**
- `TestIgnoresOtherKeys` — no change needed; tick is driven by time, not keystrokes.
- `TestViewRendersHintLine` — checks for `j/k`; still passes because the hint moves to the left side of the status bar.

---

### Implementation order

1. Write failing tests for `tickMsg` state transitions (`TestTickMsgAlwaysReissuesTicker`, `TestTickMsgBelow60sDoesNotRefresh`, `TestTickMsgAbove60sFiresRefresh`, `TestTickMsgWhenAlreadyRefreshing`, `TestTickMsgWhenNoCoins`).
2. Add `tickMsg` type, `cmdTick()`, `lastRefreshed` field — make tick tests green.
3. Update `Init()` to call `tea.Batch(m.cmdLoad(), cmdTick())` — update `TestInitReturnsBatchedCmd`.
4. Update `coinsLoadedMsg` and `pricesUpdatedMsg` handlers to set `lastRefreshed` — make `TestCoinsLoadedSetsLastRefreshed` and `TestPricesUpdatedSetsLastRefreshed` green.
5. Write failing tests for status bar display.
6. Implement `statusRight()` and `renderStatusBar()` — make status bar tests green.
7. Update `View()` to use `renderStatusBar()` and remove the full-screen error block — make `TestTableRendersWhileError` green.
8. Run `make check` — all tests pass, linter clean.

---

### Verification

```bash
make check                          # gofumpt + golangci-lint + tests (race) + govulncheck
go test -race -v ./internal/ui/...  # watch individual test output
go run ./cmd/crypto-tracker         # manual smoke: status bar shows, auto-refreshes after 60s
go run ./cmd/crypto-tracker         # kill network, verify "error: ..." in status bar, table stays visible
```

Expected test output: all existing tests pass + 14 new tests pass.
