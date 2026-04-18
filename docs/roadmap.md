# Development Roadmap

Tracer-bullet approach: each slice delivers a working vertical feature end-to-end.
Build incrementally with red-green TDD — no horizontal layers.

---

## Slice 1 — Skeleton app
STATUS: DONE

- Project structure: `cmd/crypto-tracker/main.go`, `internal/ui/app.go`
- Bubble Tea program with alt screen, renders a placeholder message
- `q` / `Ctrl+C` quits cleanly with root context cancellation
- **TDD:** model handles `tea.KeyMsg("q")` → returns `tea.Quit`

## Slice 2 — One real coin, full pipeline
STATUS: DONE

- `internal/store/store.go` (Store interface), `internal/store/sqlite.go` (SQLiteStore)
- `internal/db/db.go` + `schema.sql` (embedded, WAL + FK pragmas)
- `internal/api/coingecko.go` (CoinGeckoClient interface + HTTP implementation)
- Fetch 1 coin from `/coins/markets` → upsert into SQLite → read back → display on screen
- **TDD:** store tests with real SQLite via `t.TempDir()`, API tests with `httptest.NewServer`

## Slice 3 — Markets table: 100 coins, scrolling, formatting
STATUS: DONE

- Fetch top 100 coins on first launch; load from DB on subsequent launches
- Scrollable table: `j`/`k`/`g`/`G`, cursor highlighting
- `internal/format/format.go` — `FmtPrice`, `FmtChange` with proper thresholds
- **TDD:** format functions, cursor wrapping/clamping logic

### IMPORTANT
We have to respect the CoinGecko API's rate limiting. So we CAN'T run a request for individual coins. We MUST always update them in batches. Query the simple coin price passing all the IDs for the supported coins in a single request. Then we parse the data and update them in the database.

If you find a more efficient solution you're allowed to implement it, but ensure we never hit rate limitings when updating the coins or using the app.

## Slice 4 — Auto-refresh + status bar
STATUS: DONE

- 5s ticker → checks if 60s elapsed → fires `cmdRefresh` via `/simple/price`
- Manual refresh with `r` (no-op if already refreshing)
- Status bar states: `Synced` (green) / `Stale` (yellow, > 5 min since last refresh) / `Refreshing` (gray) / `error: <message>` (red) / `loading...` (gray)
- `staleThreshold = 5 * time.Minute` — data is considered stale after 5 minutes without a successful refresh
- Error propagation via typed `errMsg`, non-fatal — table stays visible with stale data
- **TDD:** tick/refresh state transitions, error display logic, stale detection

## Slice 5 — Tab bar + empty Portfolio tab
STATUS: DONE

- Two tabs rendered at top, `Tab`/`Shift+Tab`/`1`/`2` to switch
- Portfolio tab shows empty state: "no portfolios — press n to create one"
- Tab switching suppressed when text input is active (`InputActive()`)
- **TDD:** tab switching logic, input suppression

## Slice 6 — Create portfolio + left panel
STATUS: DONE

- Left panel lists portfolios with `▶` cursor, `j`/`k` to navigate
- `n` opens create dialog, text input (max 50 chars), `Enter` saves, `Esc` cancels
- After creation: cursor moves to new portfolio, focus enters list mode
- **TDD:** portfolio CRUD in store, dialog state transitions

## Slice 7 — Add holding: coin picker + amount input
STATUS: DONE

- `a` opens coin picker dialog (searchable, filterable list of all coins)
- Select coin → amount input → upsert holding
- Right panel shows holdings table: Coin, Ticker, Amount, Price, Value, 24h, %
- Holdings ordered by value descending, portfolio total in header
- **TDD:** holding upsert (including update-on-conflict), filter logic, computed values

## Slice 8 — List mode + edit + delete holding
STATUS: DONE

- `Enter` from menu mode enters list mode (right panel focus)
- Add focus hint in the border changing the color of the selected panel's border
- `j`/`k`/`g`/`G` in holdings list, `Esc` returns to menu
- `Enter` on holding → edit amount dialog (pre-populated amount)
- `X` on holding → delete confirmation dialog
- `PgUp`/`PgDn` -> preview scrolling from menu mode
- **TDD:** edit/delete state machine, cursor clamping after deletion

## Slice 9 - Portfolio management (edit and delete)
STATUS: DONE

When in the portfolios panel:

- `e` to edit a portfolio's name
- Open the edit dialogue box and allow user to edit the name. Renders the current name in the box
- `esc` returns to portfolios panel, `enter` saves new name and returns to portfolios panel
- Ensure portfolios' names are unique with error message and no-op editing if the name is repeated
- `X` on a portfolio opens the deletion confirmation box
- `enter` deletes the portfolio and returns to the portfolios panel
- `esc` cancels the deletion and returns to the portfolios panel
- **TDD**: ensures uniqueness, confirm the deletion and editing work

## Slice 10 — Docs & tooling cleanup
STATUS: DONE

No code changes — removes stale requirements and fixes tooling config.

- Remove terminal size guard requirement from CLAUDE.md (any terminal size is supported; the 100×30 rule is obsolete and confuses agents)
- Remove `gosimple` linter mention from CLAUDE.md (tooling compatibility issue — not applicable)
- Fix Makefile `lint` target to fail hard on golangci-lint config errors

## Slice 11 — Code correctness & clarity
STATUS: DONE

Small, targeted code changes with no new runtime behaviour.

- Fix ignored `io.ReadAll` errors in `internal/api/coingecko.go:86, 144` — `body, _ := io.ReadAll(resp.Body)` silently drops read errors in non-2xx branches
- Clean up unused `database` local in `cmd/crypto-tracker/main.go:42` — store owns the DB lifecycle; the raw `*sql.DB` reference shouldn't linger
- Consistent `defer` style for `rows.Close()` across `internal/store/sqlite.go`
- Named constants for magic numbers: 100-coin fetch limit in `internal/ui/markets.go:81, 85` and the unnamed 60s refresh threshold
- Add comment documenting quit-time cancellation assumption in `cmd/crypto-tracker/main.go:32-33, 66` (safe today due to atomic upserts; must be revisited for any future non-atomic write)
- Add comment explaining implicit message broadcast pattern in `internal/ui/app.go:62-109` (background messages fan out to both tabs via `tea.Batch`)
- **TDD:** no new tests required; verify existing suite stays green

## Slice 12 — CoinGecko rate limiting
STATUS: DONE

New runtime behaviour: the app must stay within the free-tier limit of ~30 req/min.

- Implement request throttling in `internal/api/coingecko.go`
- Inspect rate-limit response headers and back off when approaching the limit
- `r` manual refresh is a no-op if a request is already in-flight (already enforced) — extend to also no-op when rate-limited
- Surface rate-limit status in the status bar (e.g. `Rate limited — retrying in Xs`)
- **TDD:** throttle logic, backoff behaviour, status bar state for rate-limited condition


## Slice 13 — Settings tab + currency data layer
STATUS: IN_REVIEW

UI and infrastructure only — Enter does not select a currency yet.

- New DB tables: `currencies (code TEXT PK, name TEXT)` and `settings (key TEXT PK, value TEXT)`
- `CoinGeckoClient.FetchSupportedCurrencies(ctx) ([]string, error)` — new API method hitting `/simple/supported_vs_currencies`
- Filter CoinGecko response against a hardcoded fiat currency map (`internal/api/fiat.go`) — code → display name (~35 world fiat currencies; intersection with CoinGecko response determines what we store)
- `Store` gains: `UpsertCurrencies`, `GetAllCurrencies`, `GetSetting`, `SetSetting`
- On first launch (no currencies in DB): async `tea.Cmd` fires `FetchSupportedCurrencies`, filters to fiat, persists to DB; Settings tab shows "Loading currencies…" until available
- Default `selected_currency = "usd"` seeded on DB init
- New `internal/ui/settings.go`: `SettingsModel` with two modes:
  - **Browsing**: displays "Base Currency: USD" line, `Enter` opens picker, `Esc`/`q` returns to tab bar
  - **Picking**: dialog with searchable `textinput.Model`, scrollable filtered list of `Currency` rows, `j`/`k` scroll, `Esc` closes dialog (no selection), `Enter` does nothing yet
- `AppModel` gains `settings SettingsModel` and `tabSettings` (3rd tab, `tab = iota` now has 3 values)
- Tab bar renders `Markets | Portfolio | Settings`; `3` key also switches to Settings
- `InputActive()` on `SettingsModel` returns true when picker is open — suppresses tab switching
- **TDD:** store CRUD for settings/currencies, API mock for `FetchSupportedCurrencies`, settings model state transitions (browsing ↔ picking), search/filter logic, fiat filtering

## Slice 14 — Currency selection + correct price display
STATUS: PENDING

Full end-to-end: select a currency → re-fetch data in that currency → display everywhere with correct values and code.

- `CoinGeckoClient.FetchMarkets(ctx, currency, limit)` and `FetchPrices(ctx, currency, apiIDs)` — add `currency string` parameter (replaces hardcoded `"usd"`)
- `FetchPrices` response parsing uses dynamic key (`currency`) instead of `"usd"`
- `format.FmtPrice(currency, v)` — `currency` prepended as uppercase code (`USD 84,321.45` / `EUR 0.001234`)
- `format.FmtMoney(currency, v)` — same, always 2 dp (`EUR 1,500.50`)
- Picking mode `Enter` now selects the highlighted currency:
  - Writes `selected_currency` to `settings` table
  - Triggers immediate `cmdRefresh` using the new currency
  - Returns to browsing mode
- Markets tab reads selected currency from model state and passes it through `FmtPrice`/`FmtChange`
- Portfolio tab passes currency to `FmtPrice`/`FmtMoney` for Price, Value, and total columns
- Auto-refresh always uses the current `selected_currency` value from DB
- App init: read `selected_currency` from DB (default `"usd"`) before first fetch
- **TDD:** `FmtPrice`/`FmtMoney` with different currency codes, `FetchMarkets`/`FetchPrices` with `currency` param stubs, end-to-end render assertion with non-USD currency

## Slice 15 — `--debug` logging
STATUS: PENDING

- `--debug` flag enables `slog` logging to `$XDG_STATE_HOME/crypto_tracker/app.log`
- No flag → logging goes to `io.Discard`
- Directory created automatically if it doesn't exist
- **TDD:** flag parsing, log file creation, discard behaviour without flag
