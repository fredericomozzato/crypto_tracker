---
status: in_review
branch: feat/014-currency-selection-correct-price-display
---

# Slice 14 — Currency selection + correct price display

## Context

Slice 13 (DONE) delivered the Settings tab infrastructure: `currencies` and `settings` DB tables, `FetchSupportedCurrencies` API method, `Store` CRUD for settings/currencies, and a searchable currency picker dialog. The picker's `Enter` handler is a **no-op** awaiting Slice 14. All prices in Markets and Portfolio are hardcoded to USD. `FmtPrice`/`FmtMoney` hardcode the `$` symbol prefix.

This slice wires currency selection end-to-end: pick a currency → persist it → re-fetch data in that currency → display everywhere with the correct uppercase currency ticker (e.g. `USD`, `EUR`, `BRL`) as a prefix. Portfolio totals and computed values (value, proportion) must reflect the new currency after prices are updated in the DB.

## Scope

From the roadmap:

1. `FetchMarkets(ctx, currency, limit)` and `FetchPrices(ctx, apiIDs, currency)` — add `currency string` parameter
2. `FetchPrices` response parsing uses dynamic key (`currency`) instead of hardcoded `"usd"`
3. `format.FmtPrice(currency, v)` and `format.FmtMoney(currency, v)` — currency code as prefix
4. Picking mode `Enter` selects the highlighted currency: persists to DB, triggers refresh, returns to browsing
5. Markets tab reads selected currency from model state and passes it through `FmtPrice`/`FmtChange`
6. Portfolio tab passes currency to `FmtPrice`/`FmtMoney` for Price, Value, and total columns
7. Auto-refresh always uses the current `selected_currency`
8. App init reads `selected_currency` from DB (default `"usd"`) before first fetch

## Data model

**No schema changes.** The `settings` table with `selected_currency` already exists from Slice 13.

## Files to create/modify

### 1. `internal/format/format.go` — Currency-aware formatting

Always use the uppercase 3-letter currency code as the prefix — never a currency symbol. For example: `USD 84,321.45`, `EUR 0.001234`, `BRL 1,500.50`. This keeps formatting consistent and unambiguous across all currencies.

The `$` prefix currently used is replaced with the uppercase ticker. `FmtPrice` and `FmtMoney` gain a `currency string` parameter. The helper `currencyCode` lowercases the input for comparison, then returns the uppercase version:

```go
func currencyCode(currency string) string {
    return strings.ToUpper(currency)
}

func FmtPrice(v float64, currency string) string {
    prefix := currencyCode(currency)
    if v >= 1 {
        parts := strings.SplitN(fmt.Sprintf("%.2f", v), ".", 2)
        return prefix + " " + addCommas(parts[0]) + "." + parts[1]
    }
    return fmt.Sprintf("%s %.6f", prefix, v)
}

func FmtMoney(v float64, currency string) string {
    parts := strings.SplitN(fmt.Sprintf("%.2f", v), ".", 2)
    return currencyCode(currency) + " " + addCommas(parts[0]) + "." + parts[1]
}
```

`FmtChange` remains unchanged (percentages are currency-agnostic).

### 2. `internal/api/coingecko.go` — Add `currency` parameter

**Interface change:**

```go
type CoinGeckoClient interface {
    FetchMarkets(ctx context.Context, currency string, limit int) ([]store.Coin, error)
    FetchPrices(ctx context.Context, apiIDs []string, currency string) (map[string]float64, error)
    FetchSupportedCurrencies(ctx context.Context) ([]string, error)
}
```

**`FetchMarkets` implementation:** Replace `params.Set("vs_currency", "usd")` with `params.Set("vs_currency", currency)`.

**`FetchPrices` implementation:** Replace `params.Set("vs_currencies", "usd")` with `params.Set("vs_currencies", currency)`. Replace hardcoded `"usd"` key extraction with `currency`:

```go
for apiID, priceData := range apiResp {
    if price, ok := priceData[currency]; ok {
        prices[apiID] = price
    }
}
```

### 3. `internal/ui/settings.go` — Wire up currency selection

Add a new message type:

```go
type currencyChangedMsg struct {
    code string
}
```

In `settingsPicking` `KeyEnter` handler (currently a no-op on line 141), replace with:

```go
case tea.KeyEnter:
    if len(picking.filtered) == 0 {
        return m, nil
    }
    selected := picking.filtered[picking.cursor]
    if err := m.store.SetSetting(m.ctx, "selected_currency", selected.Code); err != nil {
        m.lastErr = err.Error()
        return m, nil
    }
    m.selectedCode = selected.Code
    m.mode = settingsBrowsing{}
    return m, func() tea.Msg { return currencyChangedMsg{code: selected.Code} }
```

Update `viewBrowsing()` to show the currency code:

```go
line := fmt.Sprintf("  Base Currency: %s (%s)", strings.ToUpper(m.selectedCode), currencyName)
```

### 4. `internal/ui/app.go` — Propagate currency change + init currency

Add `currency string` field to `AppModel`. Read from DB on init:

```go
func NewAppModel(ctx context.Context, s store.Store, c api.CoinGeckoClient) AppModel {
    currency, _ := s.GetSetting(ctx, "selected_currency")
    if currency == "" {
        currency = "usd"
    }
    return AppModel{
        activeTab: tabMarkets,
        markets:   NewMarketsModel(ctx, s, c, currency),
        portfolio: NewPortfolioModel(ctx, s, currency),
        settings:  NewSettingsModel(ctx, s, c),
        currency:  currency,
    }
}
```

Handle `currencyChangedMsg` in `Update` — update `m.currency`:

```go
case currencyChangedMsg:
    m.currency = msg.code
    // Falls through to broadcast, which delivers the message to all children
```

The message is already broadcast to all children via the existing fan-out (lines 115-119), so no additional dispatch needed. `MaketsModel` and `PortfolioModel` each handle it in their own `update` method.

### 5. `internal/ui/markets.go` — Currency-aware markets

Add `currency string` field to `MarketsModel`. Change constructor:

```go
func NewMarketsModel(ctx context.Context, s store.Store, c api.CoinGeckoClient, currency string) MarketsModel {
    return MarketsModel{
        ctx:      ctx,
        store:    s,
        client:   c,
        currency: currency,
    }
}
```

In `update`, add handler for `currencyChangedMsg`:

```go
case currencyChangedMsg:
    m.currency = msg.code
    m.refreshing = true
    return m, m.cmdRefresh()
```

In `cmdLoad`, change `m.client.FetchMarkets(m.ctx, coinFetchLimit)` → `m.client.FetchMarkets(m.ctx, m.currency, coinFetchLimit)`.

In `cmdRefresh`, change `m.client.FetchPrices(m.ctx, apiIDs)` → `m.client.FetchPrices(m.ctx, apiIDs, m.currency)`.

In `View()`:
- Column header: `fmt.Sprintf("Price (%s)", strings.ToUpper(m.currency))` instead of `"Price (USD)"`
- `format.FmtPrice(c.Rate, m.currency)` instead of `format.FmtPrice(c.Rate)`

### 6. `internal/ui/portfolio.go` — Currency-aware portfolio

Add `currency string` field to `PortfolioModel`. Change constructor:

```go
func NewPortfolioModel(ctx context.Context, s store.Store, currency string) PortfolioModel {
    return PortfolioModel{
        ctx:      ctx,
        store:    s,
        currency: currency,
    }
}
```

In `update`, add handlers:

```go
case currencyChangedMsg:
    m.currency = msg.code
    // Don't reload holdings yet — prices in DB are still in the old currency.
    // When MarketsModel finishes refreshing, pricesUpdatedMsg will be broadcast,
    // and we'll reload holdings then.
    return m, nil

case pricesUpdatedMsg:
    // Prices in the DB have been updated (possibly in a new currency after a
    // currency change). Reload holdings so Value, Proportion, and totals
    // reflect the new rates.
    if len(m.portfolios) > 0 {
        var cmd tea.Cmd
        m, cmd = m.update(portfolioReloadMsg{})
        return m, cmd
    }
    return m, nil
```

(Where `portfolioReloadMsg` or direct reload triggers `cmdLoadHoldings`. The exact mechanism depends on how the existing portfolio reloads — the implementation should use whatever pattern already exists for refreshing holdings data.)

In all rendering, update:
- `format.FmtPrice(h.Rate, m.currency)` instead of `format.FmtPrice(h.Rate)`
- `format.FmtMoney(h.Value, m.currency)` instead of `format.FmtMoney(h.Value)`
- `format.FmtMoney(totalValue, m.currency)` instead of `format.FmtMoney(totalValue)`
- Column headers should indicate currency: `"Price"` → includes currency context, e.g. display the currency code in the header or total line

### 7. `internal/ui/testhelpers_test.go` — Update StubAPI

Update `FetchMarkets` and `FetchPrices` signatures to match new interface:

```go
type fetchMarketsCall struct {
    limit    int
    currency string
}

func (a *StubAPI) FetchMarkets(ctx context.Context, currency string, limit int) ([]store.Coin, error) {
    a.fetchMarketsCalls = append(a.fetchMarketsCalls, fetchMarketsCall{limit: limit, currency: currency})
    if a.err != nil {
        return nil, a.err
    }
    return a.coins, nil
}

func (a *StubAPI) FetchPrices(ctx context.Context, apiIDs []string, currency string) (map[string]float64, error) {
    if a.err != nil {
        return nil, a.err
    }
    return a.prices, nil
}
```

### 8. `internal/api/coingecko_test.go` — Update tests

- **`TestFetchMarketsSuccess`**: assert `vs_currency=usd` is sent. Add subtest `TestFetchMarketsWithCurrency`: send `"eur"`, verify `vs_currency=eur` query param.
- **`TestFetchPricesSuccess`**: assert `vs_currencies=usd`. Add subtest `TestFetchPricesWithCurrency`: server returns `{"bitcoin": {"eur": 62000.00}}`, call `FetchPrices(ctx, ids, "eur")`, verify price is extracted using the `"eur"` key.
- All existing `FetchMarkets`/`FetchPrices` calls updated with `"usd"` as the currency argument.

### 9. `internal/format/format_test.go` — Update + add tests

All existing `FmtPrice(v)` calls → `FmtPrice(v, "usd")`. All existing `FmtMoney(v)` calls → `FmtMoney(v, "usd")`. Outputs change from `$` prefix to `USD ` prefix (e.g. `"$67,234.56"` → `"USD 67,234.56"`).

New tests:

| Test | Verifies |
|------|----------|
| `TestFmtPriceWithEur` | `FmtPrice(1234.56, "eur")` → `"EUR 1,234.56"` |
| `TestFmtPriceBelowOneWithEur` | `FmtPrice(0.000123, "eur")` → `"EUR 0.000123"` |
| `TestFmtPriceWithUnknownCurrency` | `FmtPrice(1234.56, "brl")` → `"BRL 1,234.56"` |
| `TestFmtMoneyWithEur` | `FmtMoney(12345.67, "eur")` → `"EUR 12,345.67"` |
| `TestFmtMoneyWithJpy` | `FmtMoney(12345.67, "jpy")` → `"JPY 12,345.67"` |
| `TestCurrencyCodeLowerCase` | `currencyCode("usd")` → `"USD"` |
| `TestCurrencyCodeUpperCase` | `currencyCode("EUR")` → `"EUR"` |

### 10. `internal/ui/app_test.go` — Add tests

| Test | Verifies |
|------|----------|
| `TestAppInitCurrencyDefault` | `NewAppModel` with no `selected_currency` setting defaults `currency` to `"usd"` |
| `TestAppInitCurrencyFromDB` | `NewAppModel` with `selected_currency=eur` sets `currency` to `"eur"` |
| `TestCurrencyChangedPropagates` | Sending `currencyChangedMsg{code: "eur"}` updates `AppModel.currency`, and the message reaches all child models |

### 11. `internal/ui/markets_test.go` — Add tests

| Test | Verifies |
|------|----------|
| `TestMarketsCurrencyChanged` | `currencyChangedMsg{code: "eur"}` sets `m.currency` to `"eur"`, `m.refreshing` to `true`, and returns a refresh cmd |
| `TestMarketsViewShowsCurrencyHeader` | After setting `m.currency = "eur"` and loading coins, `View()` contains `"Price (EUR)"` |

### 12. `internal/ui/portfolio_test.go` — Add tests

| Test | Verifies |
|------|----------|
| `TestPortfolioCurrencyChanged` | `currencyChangedMsg{code: "eur"}` sets `m.currency` to `"eur"` but does NOT immediately reload holdings |
| `TestPortfolioPricesUpdatedReloadsHoldings` | `pricesUpdatedMsg` triggers a holdings reload so totals reflect updated rates |
| `TestPortfolioViewShowsCurrency` | With `m.currency = "eur"` and holdings loaded, `View()` renders `EUR` prefix on prices and values |

### 13. `internal/ui/settings_test.go` — Add tests

| Test | Verifies |
|------|----------|
| `TestPickingEnterSelectsCurrency` | In `settingsPicking` mode, pressing `Enter` persists to store, updates `selectedCode`, transitions to `settingsBrowsing`, and returns `currencyChangedMsg` |
| `TestPickingEnterEmptyFilteredNoop` | Pressing `Enter` when `filtered` list is empty is a no-op |
| `TestPickingEnterPersistsToStore` | `SetSetting` is called with `"selected_currency"` and the selected code |

## Portfolio total correctness — key design point

When the user selects a new currency, the following sequence must occur:

1. **Settings emits `currencyChangedMsg{code: "eur"}`** after persisting to DB.
2. **`AppModel` updates `m.currency`** and broadcasts the message to all children.
3. **`MarketsModel`** receives `currencyChangedMsg` → stores `m.currency = "eur"` → triggers `cmdRefresh()` with the new currency. This fetches prices from the API in EUR and writes them to the `coins` table.
4. **`PortfolioModel`** receives `currencyChangedMsg` → stores `m.currency = "eur"` → does **NOT** reload holdings yet. The DB still has USD prices.
5. **`MarketsModel.cmdRefresh`** completes → emits `pricesUpdatedMsg`. This message is broadcast to all children.
6. **`PortfolioModel`** receives `pricesUpdatedMsg` → reloads holdings from DB. Now `coins.rate` contains EUR prices. Computed `Value = amount * rate` and `Proportion = value / total * 100` are correct in EUR.
7. All rendering uses `format.FmtPrice(rate, m.currency)` and `format.FmtMoney(value, m.currency)` with the new `"eur"` currency code.

This two-step approach ensures portfolio totals are always computed from the correct currency's prices, avoiding a race condition where holdings would be reloaded before prices are updated.

Additionally, `pricesUpdatedMsg` from auto-refresh (every 60s) also triggers a holdings reload, fixing a pre-existing issue where portfolio values went stale between refreshes.

## Implementation order

1. **`internal/format/format.go`** + **`format_test.go`** — Change `FmtPrice`/`FmtMoney` signatures, add `CurrencyPrefix`, update all tests
2. **`internal/api/coingecko.go`** + **`coingecko_test.go`** — Add `currency` param to interface + implementation, update response parsing, update all tests
3. **`internal/ui/testhelpers_test.go`** — Update `StubAPI` signatures to match new interface
4. **`internal/ui/app.go`** — Add `currency` field, read from DB in `NewAppModel`, handle `currencyChangedMsg`
5. **`internal/ui/markets.go`** — Add `currency` field, update constructor, handle `currencyChangedMsg`, update API calls, update View header + `FmtPrice` calls
6. **`internal/ui/portfolio.go`** — Add `currency` field, update constructor, handle `currencyChangedMsg` + `pricesUpdatedMsg`, update `FmtPrice`/`FmtMoney` calls + headers
7. **`internal/ui/settings.go`** — Wire up `KeyEnter` in picking mode, emit `currencyChangedMsg`
8. **`internal/ui/app_test.go`**, **`markets_test.go`**, **`portfolio_test.go`**, **`settings_test.go`** — Add/update tests per the test tables above
9. Run `make check` — all tests pass, lint clean, no race conditions

## Verification

```bash
make check
```

Expected: all tests pass, no lint errors, no race conditions, no vulnerability warnings.

Manual smoke test:

1. `go run ./cmd/crypto-tracker` — Markets tab loads in USD
2. Switch to Settings tab → Enter → pick `EUR` → Enter
3. Markets tab refreshes with EUR prices, header shows "Price (EUR)"
4. Portfolio tab shows `EUR` prefix in holdings total, price, and value columns
5. Switch back to Settings → pick `USD` → everything updates back