---
branch: feat/007-add-holding-coin-picker-amount-input
revision: 2
status: done
---

# Slice 007 — Add Holding: Coin Picker + Amount Input (Revision 2)

## Smoke test + completeness audit

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| F1 | MED | FIXED | Coin picker shows wrong error message when database has no coins |

**F1** `internal/ui/portfolio.go:290-292`

When `coinPickerReadyMsg{coins: nil}` or an empty coin list arrives, the handler checks
`len(available) == 0` and sets `m.lastErr = "all coins already in portfolio"`. However,
this condition covers two distinct cases that need different messages:

1. **No coins in the database** (Markets tab never loaded) — the PRD specifies the
   message should prompt the user to visit the Markets tab first.
2. **All coins already held** — the current message is correct for this case.

The PRD explicitly states: *"If no coins are in the database (Markets tab has never
loaded), a message prompts the user to visit the Markets tab first."* The current
implementation does not distinguish between these cases. Fix: before filtering held
coins, check `len(msg.coins) == 0` and set a distinct message such as
*"no coins loaded — visit Markets tab first"*.

## Implementation review

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| I1 | LOW | FIXED | TestCoinPickerFiltersOutAlreadyHeldCoins doesn't effectively verify filtering |

**I1** `internal/ui/portfolio_test.go:336-350`

The test creates holdings with `CoinID: 1` but uses `threeCoins()` (from `makeCoins(3)`)
which returns `Coin` structs with `ID: 0` (the `ID` field is never set). The filter
logic compares `heldCoinIDs[h.CoinID]` (keyed on `1`) against `c.ID` (all `0`), so no
coins are actually filtered out — the test still enters addCoin mode with all 3 coins
available. The assertion `!updated.InputActive()` passes for the wrong reason.

To properly test filtering, the test should set `Coin.ID` on the test data to match
the holding's `CoinID`, then verify the picker contains the correct subset (e.g. by
type-asserting the mode to `addCoin` and checking `len(mode.allCoins) == 2`).
