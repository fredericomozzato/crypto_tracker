---
status: done
branch: feat/013-settings-tab-currency-data-layer
---

# Slice 13 — Settings tab + currency data layer

## Context

Slices 1–12 are complete. The app has a working Markets tab (100 coins, auto-refresh, status bar, rate limiting) and a Portfolio tab (CRUD for portfolios and holdings). All data is hardcoded to USD. Slice 13 adds the **Settings tab** infrastructure: new DB tables for currencies and settings, a CoinGecko API method to fetch supported currencies, fiat filtering, and the Settings UI with a currency picker (browsing + picking modes). Currency **selection** (writing to settings and re-fetching in the new currency) is explicitly deferred to Slice 14 — `Enter` in the picker does nothing yet.

## Scope

1. `currencies` and `settings` DB tables via `schema.sql`
2. `Currency` type + `UpsertCurrencies` / `GetAllCurrencies` / `GetSetting` / `SetSetting` on the `Store` interface and `SQLiteStore`
3. `CoinGeckoClient.FetchSupportedCurrencies(ctx) ([]string, error)` + HTTP implementation
4. `internal/api/fiat.go` — hardcoded map of ~35 fiat currencies (code → display name)
5. On-first-launch async fetch: `FetchSupportedCurrencies` → filter against fiat map → `UpsertCurrencies`
6. `selected_currency = "usd"` seeded on DB init
7. `internal/ui/settings.go` — `SettingsModel` with `browsing` and `picking` modes
8. `AppModel` gains 3rd tab, `3` key, updated tab bar
9. `InputActive()` on `SettingsModel` suppresses tab switching when picker is open

## Data model

### `currencies` table

```sql
CREATE TABLE IF NOT EXISTS currencies (
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL
);
```

### `settings` table

```sql
CREATE TABLE IF NOT EXISTS settings (
    key TEXT NOT NULL UNIQUE,
    value TEXT NOT NULL
);
```

Default setting seeded in `db.go` `Open()`:

```sql
INSERT OR IGNORE INTO settings (key, value) VALUES ('selected_currency', 'usd');
```

### `Currency` struct (`internal/store/store.go`)

```go
type Currency struct {
    Code string
    Name string
}
```

## Files to create/modify

### `internal/db/schema.sql` — add two tables + seed

Add at end of file:

```sql
CREATE TABLE IF NOT EXISTS currencies (
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT NOT NULL UNIQUE,
    value TEXT NOT NULL
);

INSERT OR IGNORE INTO settings (key, value) VALUES ('selected_currency', 'usd');
```

### `internal/store/store.go` — add `Currency` type + 4 new interface methods

```go
type Currency struct {
    Code string
    Name string
}

type Store interface {
    // ... existing methods ...

    UpsertCurrencies(ctx context.Context, currencies []Currency) error
    GetAllCurrencies(ctx context.Context) ([]Currency, error)
    GetSetting(ctx context.Context, key string) (string, error)
    SetSetting(ctx context.Context, key, value string) error
}
```

### `internal/store/sqlite.go` — implement 4 new methods

**`UpsertCurrencies`** — inserts or updates each currency in a transaction:

```go
func (s *SQLiteStore) UpsertCurrencies(ctx context.Context, currencies []Currency) error {
    if len(currencies) == 0 {
        return nil
    }
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("beginning transaction: %w", err)
    }
    defer func() { _ = tx.Rollback() }()

    stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO currencies (code, name) VALUES (?, ?)
        ON CONFLICT(code) DO UPDATE SET name = excluded.name
    `)
    if err != nil {
        return fmt.Errorf("preparing upsert statement: %w", err)
    }
    defer func() { _ = stmt.Close() }()

    for _, c := range currencies {
        if _, err := stmt.ExecContext(ctx, c.Code, c.Name); err != nil {
            return fmt.Errorf("upserting currency %s: %w", c.Code, err)
        }
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("committing transaction: %w", err)
    }
    return nil
}
```

**`GetAllCurrencies`** — returns all currencies ordered by code:

```go
func (s *SQLiteStore) GetAllCurrencies(ctx context.Context) ([]Currency, error) {
    rows, err := s.db.QueryContext(ctx, `SELECT code, name FROM currencies ORDER BY code ASC`)
    if err != nil {
        return nil, fmt.Errorf("querying currencies: %w", err)
    }
    defer func() { _ = rows.Close() }()

    currencies := make([]Currency, 0)
    for rows.Next() {
        var c Currency
        if err := rows.Scan(&c.Code, &c.Name); err != nil {
            return nil, fmt.Errorf("scanning currency: %w", err)
        }
        currencies = append(currencies, c)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("iterating currencies: %w", err)
    }
    return currencies, nil
}
```

**`GetSetting`** — returns value or empty string if not found:

```go
func (s *SQLiteStore) GetSetting(ctx context.Context, key string) (string, error) {
    var value string
    err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
    if errors.Is(err, sql.ErrNoRows) {
        return "", nil
    }
    if err != nil {
        return "", fmt.Errorf("getting setting %q: %w", key, err)
    }
    return value, nil
}
```

**`SetSetting`** — upserts a setting:

```go
func (s *SQLiteStore) SetSetting(ctx context.Context, key, value string) error {
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO settings (key, value) VALUES (?, ?)
        ON CONFLICT(key) DO UPDATE SET value = excluded.value
    `, key, value)
    if err != nil {
        return fmt.Errorf("setting %q: %w", key, err)
    }
    return nil
}
```

### `internal/api/fiat.go` — hardcoded fiat currency map + filter function

~35 world fiat currencies with code → display name. Intersection with `/simple/supported_vs_currencies` response determines what we store.

```go
package api

var FiatCurrencies = map[string]string{
    "usd": "US Dollar",
    "eur": "Euro",
    "gbp": "British Pound",
    "jpy": "Japanese Yen",
    "aud": "Australian Dollar",
    "cad": "Canadian Dollar",
    "chf": "Swiss Franc",
    "cny": "Chinese Yuan",
    "inr": "Indian Rupee",
    "krw": "South Korean Won",
    "brl": "Brazilian Real",
    "rub": "Russian Ruble",
    "mxn": "Mexican Peso",
    "sek": "Swedish Krona",
    "nok": "Norwegian Krone",
    "dkk": "Danish Krone",
    "nzd": "New Zealand Dollar",
    "sgd": "Singapore Dollar",
    "hkd": "Hong Kong Dollar",
    "pln": "Polish Zloty",
    "thb": "Thai Baht",
    "twd": "Taiwan Dollar",
    "czk": "Czech Koruna",
    "ils": "Israeli Shekel",
    "zar": "South African Rand",
    "php": "Philippine Peso",
    "try": "Turkish Lira",
    "idr": "Indonesian Rupiah",
    "myr": "Malaysian Ringgit",
    "ars": "Argentine Peso",
    "clp": "Chilean Peso",
    "vnd": "Vietnamese Dong",
    "aed": "UAE Dirham",
    "sar": "Saudi Riyal",
}

func FilterFiat(apiCodes []string) []string {
    result := make([]string, 0, len(apiCodes))
    for _, code := range apiCodes {
        if _, ok := FiatCurrencies[code]; ok {
            result = append(result, code)
        }
    }
    return result
}
```

### `internal/api/coingecko.go` — add `FetchSupportedCurrencies`

Add to `CoinGeckoClient` interface:

```go
type CoinGeckoClient interface {
    FetchMarkets(ctx context.Context, limit int) ([]store.Coin, error)
    FetchPrices(ctx context.Context, apiIDs []string) (map[string]float64, error)
    FetchSupportedCurrencies(ctx context.Context) ([]string, error)
}
```

HTTP implementation:

```go
func (c *HTTPClient) FetchSupportedCurrencies(ctx context.Context) ([]string, error) {
    if err := c.throttle(ctx); err != nil {
        return nil, err
    }

    u := c.baseURL + "/simple/supported_vs_currencies"

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
    if err != nil {
        return nil, fmt.Errorf("creating request: %w", err)
    }

    if c.apiKey != "" {
        req.Header.Set("x-cg-demo-api-key", c.apiKey)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("fetching supported currencies: %w", err)
    }
    defer func() { _ = resp.Body.Close() }()

    if resp.StatusCode == http.StatusTooManyRequests {
        body, readErr := io.ReadAll(resp.Body)
        if readErr != nil {
            return nil, fmt.Errorf("fetching supported currencies: %d (failed to read response body: %w)", resp.StatusCode, readErr)
        }
        return nil, &RateLimitError{Body: string(body)}
    }

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        body, readErr := io.ReadAll(resp.Body)
        if readErr != nil {
            return nil, fmt.Errorf("fetching supported currencies: %d (failed to read response body: %w)", resp.StatusCode, readErr)
        }
        return nil, fmt.Errorf("fetching supported currencies: %d %s", resp.StatusCode, string(body))
    }

    var codes []string
    if err := json.NewDecoder(resp.Body).Decode(&codes); err != nil {
        return nil, fmt.Errorf("decoding response: %w", err)
    }

    return codes, nil
}
```

### `internal/ui/settings.go` — `SettingsModel`

Discriminated union for modes:

```go
type settingsMode interface{ isSettingsMode() }

type settingsBrowsing struct{}
type settingsPicking struct {
    filter   textinput.Model
    all      []store.Currency
    filtered []store.Currency
    cursor   int
}

func (settingsBrowsing) isSettingsMode() {}
func (settingsPicking) isSettingsMode()    {}
```

`SettingsModel` struct:

```go
type SettingsModel struct {
    ctx          context.Context
    store        store.Store
    client       api.CoinGeckoClient
    width        int
    height       int
    mode         settingsMode
    selectedCode string
    currencies   []store.Currency
    loading      bool
    lastErr      string
}
```

Key message types:

```go
type settingsLoadedMsg struct {
    currencies []store.Currency
    selected   string
}

type settingsNeedFetchMsg struct{}

type currenciesFetchedMsg struct {
    codes []string
}

type currenciesUpsertedMsg struct {
    currencies []store.Currency
}
```

Update logic:

- `browsing` mode:
  - `Enter` → load currencies from DB. If empty, fire `settingsNeedFetchMsg`. If available, transition to `settingsPicking`.
  - `Esc`/`q` → no-op (handled by AppModel for tab switching)
- `settingsNeedFetchMsg` → set `loading = true`, fire `cmdFetchCurrencies`
- `currenciesFetchedMsg` → filter API codes by `FilterFiat`, build `[]Currency` with names from `FiatCurrencies` map, upsert to DB, transition to `settingsPicking`
- `settingsPicking` mode:
  - `j`/`↓` → `cursor++`, clamped
  - `k`/`↑` → `cursor--`, clamped
  - `Esc` → return to `settingsBrowsing`
  - `Enter` → **no-op** (Slice 14 wires selection)
  - Other keys → forwarded to filter `textinput.Model`; re-filter list; clamp cursor

View:

- `browsing`: `"Base Currency: USD"` (using `selectedCode`), status bar hint
- `picking`: searchable dialog similar to coin picker in portfolio
- `loading`: `"Loading currencies…"` message

`InputActive()`:

```go
func (m SettingsModel) InputActive() bool {
    _, ok := m.mode.(settingsPicking)
    return ok
}
```

### `internal/ui/app.go` — add `tabSettings`, `SettingsModel`

```go
const (
    tabMarkets tab = iota
    tabPortfolio
    tabSettings
)
const tabCount = 3
```

Add `settings SettingsModel` field to `AppModel`. Update `NewAppModel` to construct `SettingsModel`. Wire into `Init`, `Update`, `View`, `renderTabBar`, `activeInputActive`.

Tab bar renders `" Markets "`, `" Portfolio "`, `" Settings "` with active highlighting.

`3` key switches to Settings tab. Tab/Shift+Tab cycle through all 3.

### `internal/ui/testhelpers_test.go` — extend `StubStore` and `StubAPI`

Add to `StubStore`:

```go
currencies []store.Currency
settings   map[string]string
```

Implement `UpsertCurrencies`, `GetAllCurrencies`, `GetSetting`, `SetSetting`.

Add to `StubAPI`:

```go
supportedCurrencies []string
```

Implement `FetchSupportedCurrencies`.

### `cmd/crypto-tracker/main.go` — no change needed

`NewAppModel` already takes `store.Store` and `api.CoinGeckoClient` — SettingsModel receives these through the constructor.

## Tests

### `internal/store/sqlite_test.go`

- **`TestUpsertCurrencies`** — insert 3 currencies, read back, verify all present and ordered by code
- **`TestUpsertCurrenciesUpdatesName`** — insert with one name, upsert with different name, verify name updated
- **`TestUpsertCurrenciesEmpty`** — verify empty slice is no-op without error
- **`TestGetAllCurrenciesEmpty`** — verify returns empty non-nil slice when no currencies
- **`TestGetSettingExisting`** — verify `GetSetting("selected_currency")` returns `"usd"` after DB init (seeded)
- **`TestGetSettingMissing`** — verify `GetSetting("nonexistent")` returns `""` without error
- **`TestSetSettingInsert`** — set a new key, read back, verify value
- **`TestSetSettingUpdate`** — set existing key to new value, read back, verify updated
- **`TestDefaultCurrencySeed`** — open fresh DB, verify `selected_currency = "usd"` exists

### `internal/api/coingecko_test.go`

- **`TestFetchSupportedCurrenciesSuccess`** — `httptest.NewServer` returning `["usd","eur","btc"]`, verify result
- **`TestFetchSupportedCurrenciesWithAPIKey`** — verify `x-cg-demo-api-key` header sent
- **`TestFetchSupportedCurrencies429`** — verify returns `*RateLimitError` on 429
- **`TestFetchSupportedCurrenciesNetworkError`** — verify network error propagated
- **`TestFetchSupportedCurrenciesContextCancelled`** — verify context cancellation respected
- **Verify `HTTPClient` implements `CoinGeckoClient`** — existing compile-time check updated to include new method

### `internal/api/fiat_test.go` (new)

- **`TestFilterFiatMatchesKnownCodes`** — verify `FilterFiat(["usd","eur","btc"])` returns only `["eur","usd"]` (sorted, btc excluded as crypto)
- **`TestFilterFiatEmptyInput`** — verify returns empty slice
- **`TestFilterFiatNoMatches`** — verify returns empty slice for all-crypto input
- **`TestFilterFiatAllFiat`** — verify all fiat codes in the map pass through

### `internal/ui/settings_test.go` (new)

- **`TestNewSettingsModel`** — verify initial state (browsing mode, no error)
- **`TestSettingsInputActiveFalseWhenBrowsing`** — verify `InputActive()` returns false
- **`TestSettingsInputActiveTrueWhenPicking`** — verify `InputActive()` returns true in picking mode
- **`TestSettingsEnterOpensPickerWhenCurrenciesAvailable`** — load currencies, press Enter, verify transition to picking mode
- **`TestSettingsEnterTriggersFetchWhenNoCurrencies`** — with empty currencies, press Enter, verify `settingsNeedFetchMsg` is produced
- **`TestSettingsPickJkNavigation`** — in picking mode, verify j/k move cursor
- **`TestSettingsPickFilterReducesList`** — type filter, verify filtered list and clamped cursor
- **`TestSettingsPickEscReturnsToBrowsing`** — verify Esc returns to browsing
- **`TestSettingsPickEnterNoOp`** — verify Enter in picking mode is a no-op (Slice 14 wires it)
- **`TestSettingsPickCursorClampsAtTop`** — k at cursor 0 stays at 0
- **`TestSettingsPickCursorClampsAtBottom`** — j at end stays at end
- **`TestSettingsBrowsingShowsSelectedCurrency`** — verify View contains the selected currency code

### `internal/ui/app_test.go`

- **`TestThreeKeySelectsSettings`** — verify pressing `3` switches to `tabSettings`
- **`TestTabBarShowsSettings`** — verify View contains "Settings"
- **`TestSettingsInputActiveSuppressesTabSwitch`** — in picking mode, Tab should not switch tabs
- **`TestTabCyclesThroughAllThreeTabs`** — verify Tab cycles Markets → Portfolio → Settings → Markets
- **`TestShiftTabCyclesBackwards`** — verify Shift+Tab cycles Settings → Portfolio → Markets → Settings

## Implementation order (TDD-first)

1. `internal/db/schema.sql` — add `currencies` and `settings` tables + seed
2. `internal/store/store.go` — add `Currency` type + 4 interface methods
3. `internal/store/sqlite.go` — implement `UpsertCurrencies`, `GetAllCurrencies`, `GetSetting`, `SetSetting`
4. `internal/store/sqlite_test.go` — write + run tests for all 4 new methods + seed test
5. `internal/api/fiat.go` — create `FiatCurrencies` map + `FilterFiat` function
6. `internal/api/fiat_test.go` — write + run filter tests
7. `internal/api/coingecko.go` — add `FetchSupportedCurrencies` to interface + HTTP implementation
8. `internal/api/coingecko_test.go` — write + run tests for `FetchSupportedCurrencies`
9. `internal/ui/settings.go` — create `SettingsModel` with browsing + picking modes
10. `internal/ui/settings_test.go` — write + run Settings model tests
11. `internal/ui/testhelpers_test.go` — extend `StubStore` + `StubAPI` with new methods
12. `internal/ui/app.go` — add `tabSettings`, `SettingsModel`, wire up
13. `internal/ui/app_test.go` — add tests for 3rd tab, tab bar, input suppression

## Verification

```bash
make check
```

All tests pass, lint is clean, no race conditions. The Settings tab shows in the tab bar, `3` switches to it, `Enter` opens the currency picker (or triggers a fetch if currencies not yet loaded), `j`/`k` navigate, typing filters, `Esc` returns to browsing, `Enter` in picker is a no-op (to be wired in Slice 14).
