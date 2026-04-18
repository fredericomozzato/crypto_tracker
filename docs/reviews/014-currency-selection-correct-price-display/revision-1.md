---
branch: feat/014-currency-selection-correct-price-display
revision: 1
status: done
---

# Slice 14 — Currency Selection + Correct Price Display (Revision 1)

## Smoke test + completeness audit

`make check` passes (fmt, lint, test, vuln — all clean). Binary builds successfully. App launches without panic.

However, **scope item 4 is not implemented**: "Picking mode Enter selects the highlighted currency" — the Enter handler in `settingsPicking` mode is still a no-op (`internal/ui/settings.go:140-142`). This makes the entire feature non-functional: the user cannot select a currency at all.

## Feature review

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| F1 | BLOCKER | FIXED | Enter key in currency picker is a no-op — selection never fires |
| F2 | HIGH | FIXED | Test `TestSettingsPickEnterNoOp` asserts no-op behavior; must be replaced with real selection tests |

**F1** `internal/ui/settings.go:140-142`

The `settingsPicking` KeyEnter handler still contains the Slice 13 placeholder:
```go
case tea.KeyEnter:
    // No-op for Slice 13 - selection handled in Slice 14
    return m, nil
```

The issue file specifies it should:
1. Guard against empty filtered list
2. Persist `selected_currency` to the settings DB via `m.store.SetSetting()`
3. Update `m.selectedCode` to the selected currency code
4. Transition to `settingsBrowsing{}`
5. Return a `currencyChangedMsg{code: selected.Code}` tea.Cmd

Without this handler, `currencyChangedMsg` is never emitted, so `AppModel.currency` is never updated, `MarketsModel` never re-fetches in the new currency, and `PortfolioModel` never reloads holdings with new prices. The entire currency-selection flow is dead.

**F2** `internal/ui/settings_test.go:196-213`

`TestSettingsPickEnterNoOp` explicitly asserts that Enter in picking mode returns `cmd == nil`. This test must be modified (or replaced) to verify real selection behavior: that `SetSetting` is called, `selectedCode` is updated, mode transitions to browsing, and a `currencyChangedMsg` is returned.

## Implementation review

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| I1 | MED | FIXED | Missing tests for currency propagation as specified in the issue |

**I1** `internal/ui/app_test.go`, `internal/ui/markets_test.go`, `internal/ui/portfolio_test.go`

The issue file specifies test tables for end-to-end currency propagation that are not implemented:
- `TestAppInitCurrencyDefault`, `TestAppInitCurrencyFromDB`, `TestCurrencyChangedPropagates` (app_test.go)
- `TestMarketsCurrencyChanged`, `TestMarketsViewShowsCurrencyHeader` (markets_test.go)
- `TestPortfolioCurrencyChanged`, `TestPortfolioPricesUpdatedReloadsHoldings`, `TestPortfolioViewShowsCurrency` (portfolio_test.go)

These tests would validate that `currencyChangedMsg` correctly updates `AppModel.currency`, triggers a refresh in MarketsModel, and causes PortfolioModel to reload holdings after prices are updated. The issue file explicitly lists them as verification checkpoints.

Once F1 is fixed, these tests should be added to provide coverage for the full currency-change flow.