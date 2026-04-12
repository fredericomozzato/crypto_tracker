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
STATUS: IN_PROGRESS

- Left panel lists portfolios with `▶` cursor, `j`/`k` to navigate
- `n` opens create dialog, text input (max 50 chars), `Enter` saves, `Esc` cancels
- After creation: cursor moves to new portfolio, focus enters list mode
- **TDD:** portfolio CRUD in store, dialog state transitions

## Slice 7 — Add holding: coin picker + amount input
STATUS: PENDING

- `a` opens coin picker dialog (searchable, filterable list of all coins)
- Select coin → amount input → upsert holding
- Right panel shows holdings table: Coin, Ticker, Amount, Price, Value, 24h, %
- Holdings ordered by value descending, portfolio total in header
- **TDD:** holding upsert (including update-on-conflict), filter logic, computed values

## Slice 8 — List mode + edit + delete holding
STATUS: PENDING

- `Enter` from menu mode enters list mode (right panel focus)
- `j`/`k`/`g`/`G` in holdings list, `Esc` returns to menu
- `Enter` on holding → edit dialog (pre-populated amount)
- `X` on holding → delete confirmation dialog
- `PgUp`/`PgDn` / `Ctrl+B`/`Ctrl+F` preview scrolling from menu mode
- **TDD:** edit/delete state machine, cursor clamping after deletion

## Slice 9 — Terminal size guard + `--debug` logging
STATUS: PENDING

- Minimum 100×30 enforced — centered message if too small, re-renders on resize
- `--debug` flag enables `slog` logging to `$XDG_STATE_HOME/crypto_tracker/app.log`
- No flag → logging goes to `io.Discard`
- **TDD:** size guard logic, flag parsing
