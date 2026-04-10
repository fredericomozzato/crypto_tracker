# Go TUI — Architecture Review

This document captures what was explored in this experimental branch: the approach
taken, what worked, what did not, and what must be done differently when the
standalone application is built from scratch.

---

## What we built

A terminal UI for the crypto tracker, written in Go (~2,100 lines across 9 files).
It connects to the CoinGecko API, caches data in a local SQLite database, and
presents two tabs: a live markets view and a portfolio manager.

### Stack

| Layer       | Library / Tool                      |
|-------------|-------------------------------------|
| TUI runtime | `charmbracelet/bubbletea` v1.3.10   |
| Styling     | `charmbracelet/lipgloss` v1.1.0     |
| Components  | `charmbracelet/bubbles` v1.0.0      |
| Database    | `modernc.org/sqlite` v1.48.2 (pure Go) |
| API         | CoinGecko REST (standard `net/http`)|

### Package layout

```
tui/
├── main.go               entry point, wiring
├── cmd_seed.go           one-off seed utility (build:ignore)
├── smoke_test.go         integration smoke tests
└── internal/
    ├── api/
    │   └── coingecko.go  HTTP client
    ├── db/
    │   ├── db.go         open + migrate
    │   └── schema.sql    embedded schema
    ├── models/
    │   ├── coin.go       coin CRUD
    │   ├── holding.go    holding CRUD + queries
    │   └── portfolio.go  portfolio CRUD
    ├── styles/
    │   └── styles.go     shared Lipgloss styles
    └── ui/
        ├── app.go        root model, tab bar
        ├── markets.go    markets tab
        └── portfolio.go  portfolio tab
```

---

## What worked well

### The Elm Architecture is correctly applied

Bubble Tea's model–update–view pattern is followed throughout. Side effects
(API calls, DB queries, timers) are always returned as `tea.Cmd` from `Update`;
no goroutines are spawned directly inside handlers. The automatic-refresh timer
uses the canonical ticker idiom: a `tickMsg` fires every 5 s, the handler checks
elapsed time, and re-issues `cmdRefresh` if the 60 s threshold is passed.

### Generic `updateChild`

```go
func updateChild[T tea.Model](m T, msg tea.Msg) (T, tea.Cmd) {
    updated, cmd := m.Update(msg)
    return updated.(T), cmd
}
```

Using a type parameter here avoids writing two nearly-identical wrapper functions.
It is a small but genuinely idiomatic use of Go generics.

### `go:embed` for the schema

```go
//go:embed schema.sql
var schema string
```

Embedding the SQL file at compile time means the binary is self-contained. The
database is created and migrated automatically on first run with no external
tooling required. Good choice for a single-binary desktop tool.

### XDG-compliant data path

`main.go` respects `XDG_DATA_HOME` and falls back to `~/.local/share`. This
is the correct convention for storing application data on Linux and is respected
by most desktop environments and packaging systems.

### SQLite pragmas

WAL mode and `PRAGMA foreign_keys = ON` are set on every connection open.
These are both correct defaults and are easy to forget; getting them in at
initialisation time is the right place.

### Upsert SQL pattern

`INSERT ... ON CONFLICT DO UPDATE SET` is used consistently for both coins and
holdings. This makes the write path idempotent without needing a prior read,
and it maps cleanly to the application's semantics (re-fetch or re-add the
same coin should update, not duplicate).

### HTTP client with explicit timeout

```go
http: &http.Client{Timeout: 15 * time.Second},
```

Using a named `http.Client` with a timeout instead of `http.DefaultClient` is
correct. `http.DefaultClient` has no timeout and will hang forever on a stalled
connection.

### Async error propagation

Errors from background commands are returned as a typed `errMsg` value and
displayed in the status bar. The UI remains responsive; the user sees the
problem without a crash.

### Smoke tests cover the storage layer independently

`TestUpsertAndRead` and `TestUpdatePrices` exercise real SQLite behaviour
without touching the UI or the network. This is the most valuable kind of test
for a project like this: it verifies the schema, query correctness, and the
upsert idempotency guarantee.

---

## What must be improved

### 1. `PortfolioModel` is a God Object

`portfolio.go` is 932 lines and the struct holds state for every possible
interaction mode simultaneously:

```go
type PortfolioModel struct {
    // main list state
    portfolios  []models.Portfolio
    holdings    []models.HoldingRow
    // add-holding sub-flow
    allCoins      []models.Coin
    filteredCoins []models.Coin
    selectedCoin  *models.Coin
    coinSearch    textinput.Model
    amountInput   textinput.Model
    // edit-holding sub-flow
    editingHolding  *models.HoldingRow
    // delete-holding sub-flow
    deletingHolding *models.HoldingRow
    // ...
}
```

All seven focus modes are handled by a long `switch` that falls through to one
of seven `handleXxxKey` functions, each with its own branching. This is hard
to read, hard to test, and will become unmanageable as new features are added.

**Fix:** Use a proper state machine with a discriminated union as the active
mode, and consider extracting each dialog into its own Bubble Tea model. Bubble
Tea supports composing models; `textinput.Model` already demonstrates this.

```go
type mode interface{ isMode() }

type browsing  struct{}
type creating  struct{ input textinput.Model }
type addCoin   struct{ search textinput.Model; filtered []models.Coin; cursor int }
type addAmount struct{ coin models.Coin; input textinput.Model }
// ...

type PortfolioModel struct {
    db         *sql.DB
    portfolios []models.Portfolio
    holdings   []models.HoldingRow
    cursor     int
    mode       mode   // only the active mode's data exists
}
```

### 2. No repository interface — raw `*sql.DB` everywhere

Every model function accepts `*sql.DB` as its first argument:

```go
func GetAllCoins(db *sql.DB) ([]Coin, error)
func UpsertHolding(db *sql.DB, ...) error
```

And the UI models store `*sql.DB` directly:

```go
type MarketsModel struct {
    db     *sql.DB
    client *api.Client
    // ...
}
```

This has three problems: the UI layer is tightly coupled to a specific storage
engine; the functions cannot be tested without a real SQLite file; and there is
no place to add caching, logging, or instrumentation without touching every
call site.

**Fix:** Define a `Store` interface in the models package (or a dedicated
`store` package) and depend on that interface in the UI layer.

```go
type Store interface {
    GetAllCoins(ctx context.Context) ([]Coin, error)
    UpsertCoin(ctx context.Context, c Coin) error
    GetHoldingsForPortfolio(ctx context.Context, id int64) ([]HoldingRow, error)
    // ...
}

type SQLiteStore struct{ db *sql.DB }

func (s *SQLiteStore) GetAllCoins(ctx context.Context) ([]Coin, error) { ... }
```

Tests can then use a stub or in-memory implementation.

### 3. No `context.Context`

None of the database queries or HTTP requests accept a `context.Context`. This
means:

- There is no way to cancel an in-flight API call when the user presses `q`.
- Database queries cannot be cancelled if they take too long.
- Timeouts must be baked into the HTTP client rather than propagated from the
  call site.

**Fix:** Follow the Go standard: `ctx context.Context` is the first parameter
of every function that performs I/O. The `tea.Cmd` closure captures the context.

```go
func cmdLoad(ctx context.Context, store Store, client *api.Client) tea.Cmd {
    return func() tea.Msg {
        coins, err := store.GetAllCoins(ctx)
        // ...
    }
}
```

A root context cancelled on quit ensures clean shutdown.

### 4. `AppModel.db` field is unused

`AppModel` stores a `*sql.DB` field but never uses it directly — the reference
is passed to child models in `NewAppModel` and then the field sits there. This
is dead weight that suggests the wiring was not fully thought through.

### 5. `isPortfolioInputActive()` breaks model encapsulation

```go
func (m AppModel) isPortfolioInputActive() bool {
    return m.activeTab == tabPortfolio &&
        (m.portfolio.focus == focusCreate || ...)
}
```

`AppModel` reads the internal `focus` field of `PortfolioModel` directly. This
couples the root model to the child model's internal state machine. If the child
model changes how it represents focus, the parent breaks.

**Fix:** `PortfolioModel` should expose a single method:

```go
func (m PortfolioModel) InputActive() bool { ... }
```

### 6. Tests use hardcoded `/tmp` paths

```go
path := "/tmp/smoke_test.db"
defer os.Remove(path)
```

Hardcoded `/tmp` paths race with each other if tests run in parallel (`go test
-parallel`), and they do not clean up on test failure before `os.Remove` runs.

**Fix:** Use `t.TempDir()`, which creates a unique directory per test and
removes it automatically when the test ends, even on failure.

```go
path := filepath.Join(t.TempDir(), "test.db")
```

### 7. Live network test is not guarded

`TestCoinGeckoAPI` makes real HTTP requests to the CoinGecko API. This makes
the test suite dependent on network availability and API rate limits. Running
`go test ./...` in CI will be flaky.

**Fix:** Skip network tests unless explicitly requested.

```go
func TestCoinGeckoAPI(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping live API test in short mode")
    }
    // ...
}
```

Run with `go test -run TestCoinGeckoAPI` or `go test -short` to control this.

### 8. `UpsertCoin` ignores `Coin.UpdatedAt`

```go
func UpsertCoin(db *sql.DB, c Coin) error {
    _, err := db.Exec(`... VALUES (?, ?, ?, ?, ?, ?, ?)`,
        c.ApiID, c.Name, c.Ticker, c.Rate, c.PriceChange, c.MarketRank,
        time.Now().Unix(),  // <-- always now, ignores c.UpdatedAt
    )
```

The `Coin` struct has an `UpdatedAt int64` field, but it is silently overridden.
A caller who sets `c.UpdatedAt` to something specific will be surprised. Either
remove the field from the struct and make the timestamp management internal to
the storage layer, or use the value from the struct.

### 9. `fmtMoney` is unexported and unreachable from outside the package

`fmtMoney` is defined in `models/holding.go` but is unexported (`fmtMoney` vs
`FmtMoney`). The UI package, which needs to format currency values, cannot
use it and likely duplicates the logic. Shared formatting helpers should live
in a well-named, exported location (e.g., a `format` package, or as an exported
function in `models`).

### 10. URL construction in `FetchPrices` is unescaped

```go
url := fmt.Sprintf(
    "%s/simple/price?ids=%s&...",
    baseURL, strings.Join(apiIDs, ","),
)
```

The IDs are joined and interpolated directly into the URL without escaping. In
practice CoinGecko IDs are URL-safe, but this is fragile. Use `url.Values` to
build query strings:

```go
params := url.Values{}
params.Set("ids", strings.Join(apiIDs, ","))
params.Set("vs_currencies", "usd")
u := baseURL + "/simple/price?" + params.Encode()
```

### 11. No retry / back-off on API errors

A transient network error or a 429 (rate limit) response will surface
immediately to the user as an error in the status bar. The next retry does not
happen until the user manually presses `r` or the 60 s auto-refresh fires.
There is no exponential back-off.

**Fix:** Implement a simple retry with back-off inside the command function,
or use a package like `github.com/cenkalti/backoff/v4`.

### 12. `updateChild` type assertion can panic

```go
return updated.(T), cmd
```

If a child model's `Update` method ever returns a different concrete type (e.g.,
a `tea.Model` wrapper), the assertion panics at runtime. Go generics do not
enforce that `Update` returns the same type; the `tea.Model` interface allows
returning any conforming type. This is safe in the current code but fragile.

### 13. Module path embeds the subdirectory

```go
module github.com/fredericomozzato/crypto_tracker/tui
```

For a standalone application that will live in its own repository, the module
path should reflect that: `github.com/fredericomozzato/crypto-tracker` (or
similar). The `/tui` suffix is an artifact of living inside the Rails project's
directory tree.

---

## Recommendations for the new application

These are the concrete things to do differently when starting the standalone
repo.

1. **Define a `Store` interface from day one.** Place it in an `internal/store`
   package. Implement it with SQLite. This decouples the UI from the database
   and makes testing straightforward.

2. **Decompose `PortfolioModel` into a state machine with sub-models.** Each
   distinct workflow (create portfolio, add holding, edit holding, delete
   holding) should be its own Bubble Tea model. The parent switches between
   them. This keeps each file small and focused.

3. **Thread `context.Context` through all I/O.** Create a root context in
   `main`, cancel it on quit. Every `tea.Cmd` that does I/O takes the context.

4. **Use `t.TempDir()` in all tests.** Delete the hardcoded `/tmp` paths.
   Guard live network tests with `testing.Short()`.

5. **Use `url.Values` to build query strings.** Never interpolate data directly
   into URL strings.

6. **Remove the `db` field from `AppModel`** once the Store interface is in
   place. The root model should only hold what it actually uses.

7. **Export `InputActive()` on child models** instead of reading their internal
   fields from the parent.

8. **Add a minimal configuration layer.** A simple config struct loaded from
   `$XDG_CONFIG_HOME/crypto_tracker/config.toml` (falling back to defaults)
   is enough to make things like the coin limit, refresh interval, and API key
   configurable without recompiling.

9. **Consider `sqlc` for query generation.** For a project of this size, writing
   raw SQL is fine, but `sqlc` generates type-safe Go functions from annotated
   SQL queries. It eliminates the scan boilerplate and keeps the SQL as the
   source of truth.

10. **Structure the repository with a `cmd/` directory** even if there is only
    one binary for now. `cmd/crypto-tracker/main.go` is the conventional entry
    point; it keeps the root of the module clean and makes it easy to add
    utilities (like the seed command) as separate binaries later.

---

## Summary

The prototype validated the core technology choices: Bubble Tea is the right
framework for a Go TUI, Lipgloss gives enough control over layout, and a local
SQLite cache makes the app fast and offline-capable. The Elm Architecture is
correctly applied at the top level.

The main thing to carry forward structurally is discipline about model size
and the dependency graph. The portfolio model accumulated too much state too
quickly. In the new app, start with the Store interface, keep each model
focused on a single screen or workflow, and add `context.Context` from the
start — it is significantly harder to retrofit than to include from day one.
