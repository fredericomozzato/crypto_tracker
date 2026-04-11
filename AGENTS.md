# AGENTS.md — crypto_tracker

Terminal UI for tracking crypto market data and managing portfolios. Built in Go
with Bubble Tea, backed by CoinGecko API and a local SQLite database.

## CRITICAL: Do not change Go version

**Never modify the `go` directive in `go.mod` without explicit human approval.**

The Go version is pinned intentionally. Changing it — even via `go mod tidy` — requires a conscious decision: it may affect all contributors, CI environments, and dependency compatibility. If a dependency or tool requires a newer Go version, **stop and ask the human** before proceeding.

---

## Key documents

Read these before implementing anything:

- [`docs/PRD.md`](docs/PRD.md) — full product spec: features, data model, keyboard
  interactions, UI layout, dialogs, status bars, and edge case behaviour.
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — lessons from the Go TUI prototype:
  what worked, what must be improved, and concrete recommendations for the new app.

---

## Stack

| Layer       | Library                              |
|-------------|--------------------------------------|
| TUI runtime | `charmbracelet/bubbletea` v1        |
| Styling     | `charmbracelet/lipgloss` v1         |
| Components  | `charmbracelet/bubbles` v1          |
| Database    | `modernc.org/sqlite` (pure Go)      |
| API         | CoinGecko REST (`net/http`)         |

- **Go version:** 1.25
- **Module path:** `github.com/fredericomozzato/crypto_tracker`
- **SQL:** Raw SQL with `go:embed` for schema. No ORM, no sqlc.
- **Platforms:** macOS and Linux. Windows is not supported.

---

## Directory structure

```
cmd/
  crypto-tracker/
    main.go            entry point, root context, wiring
internal/
  api/
    coingecko.go       HTTP client (FetchMarkets, FetchPrices)
  db/
    db.go              open + migrate
    schema.sql         embedded schema (CREATE TABLE IF NOT EXISTS)
  store/
    store.go           Store interface
    sqlite.go          SQLiteStore implementation
  ui/
    app.go             root AppModel, tab bar
    markets.go         Markets tab model
    portfolio.go       Portfolio tab model
    dialogs/           one file per dialog sub-model
  format/
    format.go          shared formatting helpers (FmtMoney, FmtPrice, etc.)
```

---

## Architecture conventions

### Program setup
The program runs in the alternate screen buffer and handles terminal resize:

```go
p := tea.NewProgram(
    model,
    tea.WithAltScreen(),
    tea.WithContext(ctx),
)
```

Every model must handle `tea.WindowSizeMsg`. If the terminal is below 100 columns × 30 rows, all content is hidden and a single centered message is rendered: `"Terminal too small — resize to at least 100×30"`.

### Elm Architecture (Bubble Tea)
- All side effects (API calls, DB queries, timers) are returned as `tea.Cmd` from `Update`. Never spawn goroutines directly inside handlers.
- Auto-refresh uses the canonical ticker idiom: a `tickMsg` fires every 5 s; the handler checks elapsed time and re-issues `cmdRefresh` if the 60 s threshold has passed.

### Store interface — required from day one
Define a `Store` interface in `internal/store/store.go`. The UI layer depends only on this interface, never on `*sql.DB` directly.

```go
type Store interface {
    GetAllCoins(ctx context.Context) ([]models.Coin, error)
    UpsertCoin(ctx context.Context, c models.Coin) error
    GetHoldingsForPortfolio(ctx context.Context, id int64) ([]models.HoldingRow, error)
    // ...
}
```

### Shutdown
Quitting (`q` or `Ctrl+C`) cancels the root context immediately. In-flight HTTP requests and DB operations are cancelled at once. No confirmation dialog. This is safe because every DB write is a single atomic upsert.

The root context is created in `main.go`, passed to `tea.NewProgram` via `tea.WithContext(ctx)`, and cancelled in a `defer` before `main` returns.

### context.Context — everywhere
`ctx context.Context` is the first parameter of every function that performs I/O (DB queries and HTTP requests). The root context is created in `main.go` and cancelled on quit. Every `tea.Cmd` closure captures the context.

### State machine for complex models
Each distinct workflow in the portfolio tab is its own Bubble Tea sub-model (create portfolio, add holding, edit holding, delete holding). The parent `PortfolioModel` holds only the active mode's data via a discriminated union:

```go
type mode interface{ isMode() }

type browsing  struct{}
type creating  struct{ input textinput.Model }
type addCoin   struct{ search textinput.Model; filtered []models.Coin; cursor int }
type addAmount struct{ coin models.Coin; input textinput.Model }

type PortfolioModel struct {
    store    store.Store
    portfolios []models.Portfolio
    holdings   []models.HoldingRow
    cursor     int
    mode       mode
}
```

### Model encapsulation
Child models expose behavior through methods, never by exposing internal fields to the parent. The root `AppModel` determines whether an input is active by calling:

```go
func (m PortfolioModel) InputActive() bool { ... }
```

### Shared formatting
All currency and percentage formatting lives in `internal/format`. Never duplicate formatting logic across packages.

- `FmtPrice(v float64) string` — `$X,XXX.XX` (2 dp, thousands separator) for ≥ $1; `$0.XXXXXX` (6 dp) for < $1
- `FmtMoney(v float64) string` — `$X,XXX.XX` for holding values
- `FmtChange(v float64) string` — `+X.XX%` / `-X.XX%`

### URL construction
Always use `url.Values` to build query strings. Never interpolate data directly into URL strings.

```go
params := url.Values{}
params.Set("ids", strings.Join(apiIDs, ","))
params.Set("vs_currencies", "usd")
u := baseURL + "/simple/price?" + params.Encode()
```

### Error propagation
Errors from background commands are returned as a typed `errMsg` value and surfaced in the status bar. The UI stays responsive; never crash on a network or DB error.

### API client interface
The CoinGecko client is defined as an interface in `internal/api`. The UI and store layers depend on this interface, never on the concrete HTTP implementation. This is what makes tests possible without network access.

```go
type CoinGeckoClient interface {
    FetchMarkets(ctx context.Context, limit int) ([]models.Coin, error)
    FetchPrices(ctx context.Context, apiIDs []string) (map[string]float64, error)
}
```

Tests use either a hand-written stub or an `httptest.NewServer` fake that returns canned JSON.

Always use a named `http.Client` with an explicit timeout (15 s) in the real implementation. Never use `http.DefaultClient`.

### Database setup
On every connection open, set:
```sql
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
```

Schema is embedded with `go:embed` and applied via `CREATE TABLE IF NOT EXISTS`. No separate migration step.

### Error wrapping
Wrap errors with context using `fmt.Errorf("...: %w", err)`. The message should read as a chain from outermost to innermost:

```go
fmt.Errorf("fetching prices: %w", err)
fmt.Errorf("upserting coin %s: %w", coin.ApiID, err)
```

This produces readable error strings (`fetching prices: request timeout`) and allows callers to unwrap with `errors.Is` / `errors.As`.

### Dependency hygiene
Run `go mod tidy` after adding or removing any dependency. Both `go.mod` and `go.sum` must be committed together. A `go.mod` that lists unused dependencies or a missing `go.sum` entry will fail CI.

### Upsert pattern
Use `INSERT ... ON CONFLICT DO UPDATE SET` for coins and holdings. Write paths are idempotent; no prior read required.

### Data directory
```
$XDG_DATA_HOME/crypto_tracker/data.db
```
Falls back to `~/.local/share/crypto_tracker/data.db`. Created automatically on first launch.

### Logging
Logging is disabled by default. Enabled with `--debug` at launch. When active, logs are written (appended) to:
```
$XDG_STATE_HOME/crypto_tracker/app.log
```
Falls back to `~/.local/state/crypto_tracker/app.log`. Directory created automatically.

Since Bubble Tea owns stdout, **never write to stdout or stderr after `tea.NewProgram` starts** — it corrupts the UI. All logging must go through the file logger. Set up the logger before starting the program:

```go
if debug {
    f, err := openLogFile() // creates XDG_STATE_HOME path
    if err == nil {
        slog.SetDefault(slog.New(slog.NewTextHandler(f, nil)))
    }
} else {
    slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
}
// then start tea.NewProgram(...)
```

---

## Testing conventions

- **No real HTTP requests.** The API client is injected via an interface; tests use a fake server (`httptest.NewServer`) or a stub that implements the interface. Never hit the live CoinGecko API in tests.
- Use `t.TempDir()` for all test database paths — never hardcode `/tmp`.
- Run `go test -race -coverprofile=coverage.out ./...` in CI — all tests must pass without network access, and the race detector must be clean.
- Storage layer tests use a real SQLite file (no mocks). UI tests use a `Store` stub.

---

## Tooling

### Formatting
- **`gofmt`** — built-in, mandatory. All code must be `gofmt`-clean.
- **`gofumpt`** — stricter superset of `gofmt` (grouped imports, blank line rules). Used in this project. Install: `go install mvdan.cc/gofumpt@latest`.

### Linting
**`golangci-lint`** — runs multiple linters in one pass. Install: `brew install golangci-lint` or see golangci-lint.run. Configured via `.golangci.yml` in the repo root.

Active linters:
- `errcheck` — catches ignored errors
- `staticcheck` — logic bugs, deprecated APIs, unreachable code
- `gosimple` — suggests simpler constructs
- `gocritic` — style and correctness
- `noctx` — flags HTTP requests made without a context

### Security
**`govulncheck`** — official Go vulnerability scanner, checks dependencies against known CVEs. Install: `go install golang.org/x/vuln/cmd/govulncheck@latest`.

### Coverage
`go test -race -coverprofile=coverage.out ./...` generates a coverage report.
View it with `go tool cover -html=coverage.out`.

---

## Distribution

Distributed via `go install`. No pre-built binaries. No Homebrew tap.

```
go install github.com/fredericomozzato/crypto_tracker/cmd/crypto-tracker@latest
```

If distribution needs grow (sharing with others), `goreleaser` is the path forward — but don't add it until there's a real need.

---

## Common commands

```bash
make check                            # fmt + lint + test + vuln (run before committing)
make fmt                              # format all files with gofumpt
make lint                             # run golangci-lint
make test                             # run tests with race detector and coverage
make vuln                             # run govulncheck
make build                            # build binary to ./crypto-tracker

go run ./cmd/crypto-tracker           # run the app
go run ./cmd/crypto-tracker --debug   # run with file logging
```

---

## Makefile

The repo ships a `Makefile` at the root. All targets are phony.

```makefile
.PHONY: fmt lint test vuln build check

fmt:
	gofumpt -w .

lint:
	golangci-lint run ./...

test:
	go test -race -coverprofile=coverage.out ./...

vuln:
	govulncheck ./...

build:
	go build -o crypto-tracker ./cmd/crypto-tracker

check: fmt lint test vuln
```

`make check` is the single command that must pass before any commit.

## golangci-lint config

`.golangci.yml` at the repo root:

```yaml
linters:
  enable:
    - errcheck
    - staticcheck
    - gosimple
    - gocritic
    - noctx

linters-settings:
  gocritic:
    enabled-tags:
      - diagnostic
      - style

issues:
  exclude-use-default: false
```

---

## Environment variables

| Variable            | Required | Purpose                                         |
|---------------------|----------|-------------------------------------------------|
| `COINGECKO_API_KEY` | No       | Demo API key for higher rate limits             |
| `XDG_DATA_HOME`     | No       | Override data directory (XDG spec)              |
| `XDG_STATE_HOME`    | No       | Override log directory (XDG spec, `--debug`)    |

## CLI flags

| Flag      | Purpose                                              |
|-----------|------------------------------------------------------|
| `--debug` | Enable file logging to `$XDG_STATE_HOME/.../app.log` |

CLI flags are parsed with the stdlib `flag` package. Do not add `cobra` or `pflag` — the app has a single command and does not need a multi-command framework.

# CoinGecko API
Whenever you have to develop any feature that touches the CoinGecko API consult the appropriate documentation endpoint:

https://docs.coingecko.com/llms-full.txt

This endpoint will provide LLM optimized documentation. ALWAYS research the endpoints to understand how to build things.

## IMPORTANT
We are running the free/demo version of the API. So we need to look at the reference for this type of application and be VERY mindful about the rate limiting that comes with it. Always develop the features trying to minimize requests and staying under the limits defined by the API wich is of ~30 calls per minute.
