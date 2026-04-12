---
status: in_review
branch: feat/008-list-mode-edit-delete-holding
---

# Slice 8 — List Mode + Edit + Delete Holding

## Context

Slices 1–7 are complete. The portfolio tab supports:
- Creating portfolios (left panel list with `j/k` navigation)
- Adding holdings (coin picker → amount input)
- Browsing mode with holdings table in the right panel

The current `PortfolioModel` uses a discriminated union `portfolioMode` with four states: `browsing`, `creating`, `addCoin`, `addAmount`. What's missing is **list mode** (entering the holdings list to select/edit/delete holdings), **edit amount dialog**, **delete confirmation dialog**, **preview scrolling** from menu mode, and **focused panel border highlighting**.

## Scope

Per the roadmap:
- `Enter` from menu mode enters list mode (right panel focus)
- Add focus hint in the border changing the color of the selected panel's border
- `j`/`k`/`g`/`G` in holdings list, `Esc` returns to menu
- `Enter` on holding → edit amount dialog (pre-populated amount)
- `X` on holding → delete confirmation dialog
- `PgUp`/`PgDn` → preview scrolling from menu mode
- TDD: edit/delete state machine, cursor clamping after deletion

## Data Model

No SQL schema changes. `holdings` table already supports `DELETE` and `UpsertHolding` updates the amount on conflict. The `Store` interface already has `DeleteHolding` and `UpsertHolding`.

## Files to Create/Modify

### 1. `internal/ui/portfolio.go` — Major modifications

**New mode types** (add to existing discriminated union):

```go
type listing struct{} // list mode: right panel focus, j/k navigate holdings

type editingAmount struct {
    holding   store.HoldingRow
    input     textinput.Model
    errMsg    string
    listMode  listing // preserved so Esc returns to list mode with state intact
}

type deleting struct {
    holding   store.HoldingRow
    listMode  listing // preserved so Esc/cancel returns to list mode with state intact
}
```

Add `isPortfolioMode()` implementations:

```go
func (listing) isPortfolioMode()      {}
func (editingAmount) isPortfolioMode() {}
func (deleting) isPortfolioMode()      {}
```

**New fields on `PortfolioModel`:**

```go
type PortfolioModel struct {
    // ... existing fields ...
    holdingsCursor int    // cursor position within holdings list (list mode)
    scrollOffset   int    // vertical scroll offset for holdings list
}
```

**New message types:**

```go
type holdingDeletedMsg struct {
    holdings []store.HoldingRow
}
```

**New methods on `PortfolioModel`:**

```go
func (m PortfolioModel) cmdDeleteHolding(portfolioID, holdingID int64) tea.Cmd
func (m PortfolioModel) cmdUpdateHoldingAmount(portfolioID, coinID int64, amount float64) tea.Cmd
```

`cmdDeleteHolding` calls `store.DeleteHolding` then `store.GetHoldingsForPortfolio`, returning `holdingDeletedMsg`.

`cmdUpdateHoldingAmount` calls `store.UpsertHolding` (reuses existing upsert — same amount update-on-conflict semantics) then `store.GetHoldingsForPortfolio`, returning `holdingsSavedMsg`.

**Key handler changes in `update`:**

In `browsing` mode:
- `Enter` key: if `len(m.holdings) > 0`, transition to `listing` mode with `holdingsCursor = 0`, `scrollOffset = 0`
- `PgDn` / `Ctrl+F`: increment scroll offset by half viewport height
- `PgUp` / `Ctrl+B`: decrement scroll offset by half viewport height

In `listing` mode:
- `j`/`↓`: `holdingsCursor++`, clamp, adjust scroll
- `k`/`↑`: `holdingsCursor--`, clamp to 0
- `g`: `holdingsCursor = 0`, adjust scroll
- `G`: `holdingsCursor = len(m.holdings) - 1`, adjust scroll
- `Enter`: if cursor valid, transition to `editingAmount` mode with pre-populated input (current amount, 4 decimal places)
- `X`: if cursor valid, transition to `deleting` mode
- `a`: open coin picker (same as from browsing — reuse existing `cmdOpenCoinPicker` logic, but preserve `listing` state to return to after save)
- `Esc`: return to `browsing` mode

In `editingAmount` mode:
- `Enter`: validate amount > 0, call `cmdUpdateHoldingAmount`, return to `listing` on success message
- `Esc`: return to `listing` mode
- Any other key: delegate to input

In `deleting` mode:
- `Enter`: call `cmdDeleteHolding`, on success set mode to `listing` (or `browsing` if holdings now empty)
- `Esc`: return to `listing` mode
- All other keys silently ignored

**Message handler changes:**

- `holdingDeletedMsg`: update `m.holdings`, clamp `holdingsCursor` (if deleted was last, clamp to new length - 1), set mode to `listing` (or `browsing` if empty)
- `holdingsSavedMsg`: when coming from edit amount mode, set mode back to `listing` and re-clamp cursor

**`InputActive()` update:**

```go
func (m PortfolioModel) InputActive() bool {
    switch m.mode.(type) {
    case creating, addCoin, addAmount, editingAmount:
        return true
    }
    return false
}
```

Note: `listing` and `deleting` are NOT input-active — `deleting` blocks all other keys, `listing` uses j/k navigation.

**View changes:**

- `listing` mode: render two-panel layout with right panel border highlighted (different color)
- `browsing` mode: render two-panel layout with left panel border highlighted
- `editingAmount` mode: centered dialog overlay with coin name, ticker, current amount, input
- `deleting` mode: centered dialog overlay with coin name, ticker, current amount, "Enter to delete / Esc to cancel"
- Status bar: update hints for each new mode

**Scroll offset for preview scrolling (browsing mode):**

- New field `scrollOffset int` on `PortfolioModel`
- `PgDn`/`Ctrl+F`: `scrollOffset += visibleRows/2`, clamped to max offset
- `PgUp`/`Ctrl+B`: `scrollOffset -= visibleRows/2`, clamped to 0
- Reset `scrollOffset` to 0 when navigating portfolios with j/k

**Holdings cursor adjustments in `listing` mode:**

New helper methods:
```go
func (m PortfolioModel) visibleHoldingsRows() int  // computes how many holdings can display
func (m *PortfolioModel) adjustHoldingsViewport()    // ensures holdingsCursor stays visible
```

**Border highlighting:**

Determine the "focused" panel based on mode:
- `browsing`: left panel focused (accent border), right panel unfocused (dim border)
- `listing`: right panel focused (accent border), left panel unfocused (dim border)
- Dialog modes (`creating`, `addCoin`, `addAmount`, `editingAmount`, `deleting`): no focus change (last known focus retained — for dialogs from list mode, right panel stays focused; from browsing, left panel)

### 2. `internal/ui/portfolio_test.go` — New tests

**State machine tests:**

- `TestEnterFromBrowsingToListingMode`: pressing Enter with holdings transitions from `browsing` to `listing`, sets `holdingsCursor = 0`
- `TestEnterFromBrowsingNoHoldingsIsNoOp`: pressing Enter with no holdings is a no-op
- `TestListingJkNavigation`: j/k moves `holdingsCursor` in holdings
- `TestListingGJumpsToTop`: `g` moves `holdingsCursor` to 0
- `TestListingGJumpsToBottom`: `G` moves `holdingsCursor` to last holding
- `TestListingClampsAtTop`: `k` at top stays at 0
- `TestListingClampsAtBottom`: `j` at bottom stays at last index
- `TestListingEscReturnsToBrowsing`: Esc from listing returns to browsing
- `TestListingEnterOpensEditDialog`: Enter on a holding in list mode transitions to `editingAmount`, input is pre-populated with current amount
- `TestListingXOpensDeleteDialog`: `X` on a holding transitions to `deleting` mode
- `TestListingAOpensCoinPicker`: `a` from listing mode opens coin picker

**Edit amount dialog tests:**

- `TestEditAmountEscReturnsToListing`: Esc from edit dialog returns to listing mode
- `TestEditAmountEnterWithEmptyNoOp`: empty input is no-op
- `TestEditAmountEnterWithNonNumericShowsError`: non-numeric input shows error message
- `TestEditAmountEnterWithZeroOrNegativeShowsError`: zero/negative input shows error
- `TestEditAmountEnterWithValidAmountReturnsCmd`: valid amount returns a command
- `TestEditingAmountInputActive`: editingAmount mode reports InputActive() = true

**Delete confirmation tests:**

- `TestDeleteConfirmEscReturnsToListing`: Esc from delete dialog returns to listing
- `TestDeleteConfirmEnterReturnsCmd`: Enter on delete dialog returns a command
- `TestDeletingInputActive`: deleting mode reports InputActive() = false (blocks tab switching but text input isn't active)
- `TestDeleteConfirmOtherKeysIgnored`: any key other than Enter/Esc is ignored in delete mode

**Cursor clamping after deletion:**

- `TestCursorClampedAfterDelete`: after deletion, if holdingsCursor >= len(holdings), it's clamped to the new max
- `TestCursorStaysAtSamePositionAfterDelete`: if cursor index still valid after delete, it stays

**Preview scrolling tests:**

- `TestBrowsingPgDnScrollsHoldingsPreview`: PgDn in browsing mode increments scrollOffset
- `TestBrowsingPgUpScrollsHoldingsPreview`: PgUp in browsing mode decrements scrollOffset
- `TestBrowsingPgUpDoesNotGoBelowZero`: PgUp when scrollOffset is 0 stays at 0
- `TestBrowsingJkResetsScrollOffset`: pressing j/k in browsing resets scrollOffset to 0

**HoldingsSavedMsg from edit returns to listing:**

- `TestHoldingsSavedFromEditReturnsToListing`: after saving via edit, mode goes back to listing (not browsing)

**holdingDeletedMsg tests:**

- `TestHoldingDeletedMsgUpdatesHoldings`: holdingDeletedMsg replaces holdings slice
- `TestHoldingDeletedMsgClampsCursor`: if cursor was at deleted position, it's clamped
- `TestHoldingDeletedMsgReturnsToListing`: after delete, mode is listing (or browsing if empty)
- `TestHoldingDeletedMsgToBrowsingWhenEmpty`: if all holdings deleted, mode returns to browsing

**View tests:**

- `TestListingModeShowsPanelFocus`: listing mode renders right panel with accent border
- `TestBrowsingModeShowsPanelFocus`: browsing mode renders left panel with accent border
- `TestEditDialogShowsCoinName`: edit amount dialog shows the holding's coin name and ticker
- `TestDeleteDialogShowsCoinName`: delete confirmation shows coin name, ticker, and amount

## Implementation Order

1. **Add `listing`, `editingAmount`, `deleting` mode types** — add type definitions and `isPortfolioMode()` methods to `portfolio.go`
2. **Add `holdingsCursor` and `scrollOffset` fields** to `PortfolioModel`
3. **Write tests for state transitions** — `portfolio_test.go`: all the "Test" functions listed above for entering/exiting list mode, edit, and delete
4. **Implement `browsing` mode key handlers** — Enter transitions to listing, PgDn/PgUp for scroll preview
5. **Implement `listing` mode `update` handler** — j/k/g/G navigation, Enter to edit, X to delete, a for add, Esc to browsing
6. **Implement `editingAmount` mode handler** — input delegation, validation, cmd dispatch
7. **Implement `deleting` mode handler** — Enter confirm, Esc cancel
8. **Add `holdingDeletedMsg` type and handler** — update holdings, clamp cursor, return to listing/browsing
9. **Update `holdingsSavedMsg` handler** — when from edit mode, return to listing instead of browsing
10. **Implement `cmdDeleteHolding` and `cmdUpdateHoldingAmount`**
11. **Update `View()`** — panel border highlighting, edit/delete dialogs, listing-mode cursor highlighting in holdings table, status bar hints for new modes
12. **Update `InputActive()`** — add `editingAmount` to true cases, `deleting` stays false
13. **Implement preview scrolling** — PgDn/Ctrl+F and PgUp/Ctrl+B in browsing mode
14. **Run tests** — `go test -race -coverprofile=coverage.out ./...`

## Verification

```bash
make check
```

Expected: all tests pass, lint clean, no race conditions, no vulnerabilities.

Specific test commands:
```bash
go test -race -run TestPortfolio -v ./internal/ui/
go test -race -run TestListing -v ./internal/ui/
go test -race -run TestEdit -v ./internal/ui/
go test -race -run TestDelete -v ./internal/ui/
go test -race -run TestBrowsing -v ./internal/ui/
```
