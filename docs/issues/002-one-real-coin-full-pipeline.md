---
status: pending
branch: feat/002-one-real-coin-full-pipeline
---

# Slice 2 — One Real Coin, Full Pipeline

## Context

Slice 1 delivered the skeleton: `cmd/crypto-tracker/main.go`, `internal/ui/app.go`,
root context, alt screen, and quit handling. The app currently renders a centered
placeholder string and does nothing else.

This slice builds the full vertical pipeline for the first time: HTTP fetch →
SQLite storage → screen render. Only one coin is needed to prove the pipeline
works end-to-end. The goal is to have every layer (API client, database, store,
UI wiring) in place and tested, so that Slice 3 can scale up to 100 coins
without touching the plumbing.

## Scope (from roadmap + additions)

- `internal/store/store.go` (Store interface), `internal/store/sqlite.go` (SQLiteStore)
- `internal/db/db.go` + `schema.sql` (embedded, WAL + FK pragmas)
- `internal/api/coingecko.go` (CoinGeckoClient interface + HTTP implementation)
- Fetch 1 coin from `/coins/markets` → upsert into SQLite → read back → display
- **`r` key triggers a manual price refresh** via `/simple/price` → update store
  → re-render. This proves the update path (not just the initial load) and
  exercises both API endpoints end-to-end.
- **TDD:** store tests with real SQLite via `t.TempDir()`, API tests with `httptest.NewServer`

## Data model

From the PRD, the `coins` table:

```sql
CREATE TABLE IF NOT EXISTS coins (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    api_id      TEXT    NOT NULL UNIQUE,
    name        TEXT    NOT NULL,
    ticker      TEXT    NOT NULL,
    rate        REAL    NOT NULL DEFAULT 0,
    price_change REAL   NOT NULL DEFAULT 0,
    market_rank INTEGER NOT NULL DEFAULT 0,
    updated_at  INTEGER NOT NULL DEFAULT 0
);
```

Portfolios and holdings tables are not needed yet — they arrive in Slices 6–7.

## Files to create

### 1. `internal/db/schema.sql`

- `coins` table only (as above)
- `CREATE TABLE IF NOT EXISTS` so it's idempotent

### 2. `internal/db/db.go`

- `//go:embed schema.sql` to embed the schema at compile time
- `func Open(path string) (*sql.DB, error)`:
  - Opens SQLite via `modernc.org/sqlite` driver
  - Sets `PRAGMA journal_mode = WAL`
  - Sets `PRAGMA foreign_keys = ON`
  - Executes embedded schema
  - Returns the `*sql.DB`
- Creates the parent directory if it doesn't exist (`os.MkdirAll`)

### 3. `internal/store/store.go`

- Defines the `Store` interface (only the methods needed for this slice):

```go
type Store interface {
    UpsertCoin(ctx context.Context, c Coin) error
    GetAllCoins(ctx context.Context) ([]Coin, error)
    UpdatePrices(ctx context.Context, prices map[string]float64) error
    Close() error
}
```

- Defines the `Coin` struct here (not in a separate `models` package — keep it
  close to the interface that uses it until there's a reason to split):

```go
type Coin struct {
    ID          int64
    ApiID       string
    Name        string
    Ticker      string
    Rate        float64
    PriceChange float64
    MarketRank  int
    UpdatedAt   int64
}
```

### 4. `internal/store/sqlite.go`

- `SQLiteStore` struct holding `*sql.DB`
- `func NewSQLiteStore(db *sql.DB) *SQLiteStore`
- Implements `Store`:
  - `UpsertCoin`: `INSERT INTO coins ... ON CONFLICT(api_id) DO UPDATE SET ...`
    — sets `updated_at` to `time.Now().Unix()` (the store owns the timestamp,
    not the caller)
  - `GetAllCoins`: `SELECT ... FROM coins ORDER BY market_rank ASC`
  - `UpdatePrices(ctx, map[string]float64)`: takes a map of `api_id → price`,
    updates `rate` and `updated_at` for each matching coin in a single
    transaction. This is the write path for `/simple/price` refreshes.
  - `Close`: closes the underlying `*sql.DB`

### 5. `internal/store/sqlite_test.go`

Tests use a real SQLite database via `t.TempDir()`. No mocks.

1. **TestUpsertAndReadBack** — upsert a coin, read all coins, assert fields match
2. **TestUpsertUpdatesExisting** — upsert same `api_id` twice with different
   price, assert only one row exists and the price is the latest value
3. **TestGetAllCoinsOrdering** — upsert 3 coins with different `market_rank`,
   assert `GetAllCoins` returns them sorted by rank ascending
4. **TestGetAllCoinsEmpty** — no coins inserted, `GetAllCoins` returns empty
   slice (not nil), no error
5. **TestUpdatePrices** — upsert 2 coins, call `UpdatePrices` with new prices
   for both, read back, assert `Rate` and `UpdatedAt` changed
6. **TestUpdatePricesUnknownCoin** — call `UpdatePrices` with an `api_id` that
   doesn't exist in the DB, assert no error (silently ignored)

### 6. `internal/api/coingecko.go`

- Defines the `CoinGeckoClient` interface:

```go
type CoinGeckoClient interface {
    FetchMarkets(ctx context.Context, limit int) ([]store.Coin, error)
    FetchPrices(ctx context.Context, apiIDs []string) (map[string]float64, error)
}
```

  Both methods are needed in this slice: `FetchMarkets` for the initial load,
  `FetchPrices` for the `r` refresh command.

- `HTTPClient` struct (the concrete implementation):
  - Holds `*http.Client` (with 15 s timeout) and `baseURL string`
  - `func NewHTTPClient(apiKey string) *HTTPClient`
    - Uses `https://api.coingecko.com/api/v3` as base URL
    - If `apiKey` is non-empty, uses `https://pro-api.coingecko.com/api/v3`
      and sets the `x-cg-demo-api-key` header (or query param, per CoinGecko
      demo key docs)
  - `FetchMarkets(ctx context.Context, limit int) ([]store.Coin, error)`:
    - Builds URL with `url.Values`: `vs_currency=usd`, `order=market_cap_desc`,
      `per_page=<limit>`, `page=1`, `sparkline=false`,
      `price_change_percentage=24h`
    - Creates request with `http.NewRequestWithContext(ctx, ...)`
    - Parses JSON response into `[]store.Coin`
    - Wraps errors with `fmt.Errorf("fetching markets: %w", err)`
    - On non-2xx status: reads the body and returns
      `fmt.Errorf("fetching markets: %d %s", status, body)`
  - `FetchPrices(ctx context.Context, apiIDs []string) (map[string]float64, error)`:
    - Builds URL with `url.Values`: `ids=<comma-joined>`, `vs_currencies=usd`
    - Returns `map[string]float64` where key is the API ID and value is the
      USD price (e.g. `{"bitcoin": 67000.00}`)
    - Same error handling pattern as `FetchMarkets`

### 7. `internal/api/coingecko_test.go`

Uses `httptest.NewServer` — no real HTTP requests.

**FetchMarkets tests:**

1. **TestFetchMarketsSuccess** — fake server returns valid JSON for 1 coin,
   assert the returned `[]store.Coin` has correct fields
2. **TestFetchMarketsAPIError** — fake server returns 429 with error body,
   assert error message includes status code and body
3. **TestFetchMarketsNetworkError** — use a server that closes immediately,
   assert a non-nil error is returned
4. **TestFetchMarketsContextCancelled** — cancel context before request,
   assert `context.Canceled` error

**FetchPrices tests:**

5. **TestFetchPricesSuccess** — fake server returns
   `{"bitcoin":{"usd":67000}}`, assert map contains `"bitcoin" → 67000.0`
6. **TestFetchPricesAPIError** — fake server returns 429, assert error
   includes status code

### 8. `internal/ui/app.go` (modify)

- `AppModel` gains a `store.Store` and `api.CoinGeckoClient` dependency
- `NewAppModel(s store.Store, c api.CoinGeckoClient) AppModel`
- `Init() tea.Cmd` — returns a command that fetches 1 coin from the API,
  upserts it into the store, reads it back, and returns it as a `coinsLoadedMsg`
- New message types:
  - `coinsLoadedMsg { coins []store.Coin }`
  - `pricesUpdatedMsg { coins []store.Coin }` — returned after a refresh
  - `errMsg { err error }`
- New fields on `AppModel`:
  - `coins []store.Coin` — the loaded coin data
  - `err string` — last error message (empty = no error)
  - `refreshing bool` — true while a refresh is in flight (prevents double `r`)
- `Update` handles:
  - `tea.KeyMsg` `r` → if not already refreshing: set `refreshing = true`,
    return a `tea.Cmd` that calls `FetchPrices` with the API IDs of all loaded
    coins, then `UpdatePrices` on the store, then `GetAllCoins` to reload,
    returning a `pricesUpdatedMsg` (or `errMsg` on failure)
  - `coinsLoadedMsg` — stores the coins, clears error
  - `pricesUpdatedMsg` — stores the updated coins, clears `refreshing` and error
  - `errMsg` — stores the error string, clears `refreshing`
- `View` renders:
  - If coins are loaded: display the coin's name, ticker, price, and 24h change
    as a simple formatted string (not a table yet — that's Slice 3). Include
    `"r to refresh"` in the hint text and `"refreshing..."` when in flight.
  - If error: display the error
  - If neither: display `"loading..."`

### 9. `internal/ui/app_test.go` (modify)

Update existing tests for new constructor signature (pass nil or stub
dependencies where not exercised). Add:

1. **TestCoinsLoadedMsg** — send a `coinsLoadedMsg` with one coin, assert
   `View()` contains the coin name and price
2. **TestErrMsg** — send an `errMsg`, assert `View()` contains the error text
3. **TestInitReturnsCmd** — assert `Init()` returns a non-nil command
4. **TestRefreshKeyReturnsCmdWhenCoinsLoaded** — load coins first via
   `coinsLoadedMsg`, then send `r` key, assert a non-nil command is returned
   and `refreshing` is true
5. **TestRefreshKeyIgnoredWhenAlreadyRefreshing** — set `refreshing = true`,
   send `r` key, assert nil command (no double refresh)
6. **TestRefreshKeyIgnoredWhenNoCoins** — send `r` before any coins are
   loaded, assert nil command (nothing to refresh)
7. **TestPricesUpdatedMsg** — send `pricesUpdatedMsg` with updated coins,
   assert `View()` shows new price and `refreshing` is cleared

### 10. `cmd/crypto-tracker/main.go` (modify)

- Import and open the database via `db.Open(path)` where `path` uses
  `XDG_DATA_HOME` logic (same pattern as `logFilePath` but for data)
- Create `store.NewSQLiteStore(database)`
- Read `COINGECKO_API_KEY` from environment
- Create `api.NewHTTPClient(apiKey)`
- Pass both to `ui.NewAppModel(s, c)`
- Defer `s.Close()`

## Dependencies to add

- `modernc.org/sqlite` — pure-Go SQLite driver (no CGo)

Install: `go get modernc.org/sqlite`

## Implementation order

1. Write `internal/db/schema.sql`
2. Write `internal/db/db.go` — embed schema, Open function
3. Write `internal/store/store.go` — Coin struct, Store interface
4. Write `internal/store/sqlite_test.go` (all tests, all red)
5. Write `internal/store/sqlite.go` — make store tests green
6. Write `internal/api/coingecko_test.go` (all tests, all red)
7. Write `internal/api/coingecko.go` — make API tests green
8. Update `internal/ui/app_test.go` — add new tests, fix constructor calls
9. Update `internal/ui/app.go` — add dependencies, Init command, new messages
10. Update `cmd/crypto-tracker/main.go` — wire DB, store, API client
11. Run `make check` — all must pass
12. Manual smoke test: `go run ./cmd/crypto-tracker` — should show one coin's data

## CoinGecko API response shapes

### `/coins/markets` (initial load)

```json
[
  {
    "id": "bitcoin",
    "symbol": "btc",
    "name": "Bitcoin",
    "current_price": 67000.00,
    "price_change_percentage_24h": -1.23,
    "market_cap_rank": 1
  }
]
```

Map to `store.Coin`:
- `id` → `ApiID`
- `symbol` (uppercased) → `Ticker`
- `name` → `Name`
- `current_price` → `Rate`
- `price_change_percentage_24h` → `PriceChange`
- `market_cap_rank` → `MarketRank`

### `/simple/price` (refresh)

```json
{
  "bitcoin": {
    "usd": 67123.45
  }
}
```

Map to `map[string]float64`: key is the coin's `api_id`, value is the USD price.
Passed to `Store.UpdatePrices` to batch-update the `rate` column.

## Verification

```bash
make check              # fmt + lint + test + vuln — must all pass
make build              # produces ./crypto-tracker binary
go run ./cmd/crypto-tracker  # fetches bitcoin, displays name + price
                             # press r — price refreshes via /simple/price
                             # press q — quits cleanly
```

After this slice, the full pipeline is proven end-to-end: initial load
(API → Store → UI) and refresh (key → API → Store → UI). Slice 3 scales
to 100 coins and adds the scrollable table. Slice 4 adds the auto-refresh
ticker but the manual `r` refresh already works.
