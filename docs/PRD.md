# Crypto Tracker TUI — Product Requirements Document

This document describes the features implemented in the experimental Go TUI
prototype. It is intended to be read alongside `ARCHITECTURE.md` when building
the standalone application from scratch.

Presentation details (colors, borders, exact column widths) are intentionally
omitted — those will be iterated based on user feedback. This document covers
**what the app does**, **how the user interacts with it**, and **what data it
manages**.

---

## Overview

A standalone terminal application for tracking cryptocurrency market data and
managing personal portfolios. It runs entirely on the local machine: market data
is fetched from the CoinGecko public API and cached in a local SQLite database.
No account, no server, no sync.

### Core principles

- **Offline-first after seed.** Once the coin list is fetched once, the app
  works without a network connection. Prices go stale but nothing breaks.
- **Single binary.** No external runtime, no config files required to start.
- **Keyboard-driven.** Every action is reachable from the keyboard. No mouse
  required.
- **Local data only.** Portfolios and holdings are stored in a SQLite file in
  the user's data directory. Nothing is sent anywhere.

---

## Data model

### Coins

A coin is a cryptocurrency tracked from the CoinGecko API. It is stored locally
once fetched and updated on each refresh.

| Field         | Type    | Description                          |
|---------------|---------|--------------------------------------|
| `id`          | integer | Local primary key                    |
| `api_id`      | text    | CoinGecko identifier (e.g. `bitcoin`) — unique |
| `name`        | text    | Display name (e.g. `Bitcoin`)        |
| `ticker`      | text    | Symbol, uppercased (e.g. `BTC`)      |
| `rate`        | real    | Current price in USD                 |
| `price_change`| real    | 24 h price change as a percentage    |
| `market_rank` | integer | CoinGecko market cap rank            |
| `updated_at`  | integer | Unix timestamp of last price update  |

### Portfolios

A named collection of holdings owned by the user. A user can have multiple
portfolios (e.g. "Long Term", "Trading").

| Field        | Type    | Description              |
|--------------|---------|--------------------------|
| `id`         | integer | Primary key              |
| `name`       | text    | User-defined name        |
| `created_at` | integer | Unix timestamp           |

### Holdings

A specific coin held inside a portfolio, with a quantity.

| Field          | Type    | Description                                           |
|----------------|---------|-------------------------------------------------------|
| `id`           | integer | Primary key                                           |
| `portfolio_id` | integer | Foreign key → portfolios (CASCADE DELETE)             |
| `coin_id`      | integer | Foreign key → coins (CASCADE DELETE)                  |
| `amount`       | real    | Quantity owned (e.g. `0.5` for 0.5 BTC)              |

The pair `(portfolio_id, coin_id)` is unique: a coin can appear only once per
portfolio. Adding the same coin again updates the amount (upsert).

**Computed at query time** (not stored):

- `value` = `amount × rate`
- `proportion` = `value / portfolio_total × 100`
- `portfolio_total` = sum of all holding values in the portfolio

---

## External dependency

**CoinGecko REST API** (`https://api.coingecko.com/api/v3`)

Two endpoints are used:

1. **`/coins/markets`** — fetches the top N coins by market cap, with name,
   ticker, current price, 24 h change, and market rank. Used once on first
   launch to seed the local database.

2. **`/simple/price`** — fetches current prices for a given list of coin IDs.
   Used on every subsequent refresh (avoids re-fetching metadata).

An optional API key (`COINGECKO_API_KEY` environment variable) is supported for
higher rate limits. The app works without it on the free tier.

---

## Logging

Logging is disabled by default. It is enabled by passing the `--debug` flag at
launch:

```
crypto-tracker --debug
```

When enabled, logs are written to:

```
$XDG_STATE_HOME/crypto_tracker/app.log
```

If `XDG_STATE_HOME` is not set, defaults to `~/.local/state/crypto_tracker/app.log`.
The directory is created automatically. The log file is appended to across runs.

When `--debug` is not set, nothing is written anywhere.

---

## Shutdown behavior

Pressing `q` or `Ctrl+C` quits immediately. Any in-flight network requests or
database writes are cancelled at once. No confirmation is shown. Because every
database write is a single atomic upsert, there is no risk of partial or corrupt
data on exit.

---

## Installation

```
go install github.com/fredericomozzato/crypto_tracker/cmd/crypto-tracker@latest
```

Requires Go 1.23+. No pre-built binaries are provided.

---

## Supported platforms

- macOS
- Linux

Windows is not supported.

---

## Display mode

The app runs in the terminal's **alternate screen buffer**. On launch it takes
over the full terminal window; on exit the previous shell output is fully
restored. The TUI never mixes into the shell scroll history.

---

## Terminal requirements

The app requires a minimum terminal size of **100 columns × 30 rows**.

If the terminal is smaller than this, all content is hidden and a single centered
message is shown:

```
Terminal too small — resize to at least 100×30
```

The UI re-renders automatically once the terminal is resized to meet the minimum.

---

## Application structure

The app has two tabs. The tab bar is always visible at the top.

```
[ Markets ]  [ Portfolio ]
```

Switching tabs:

| Key           | Action                    |
|---------------|---------------------------|
| `Tab`         | Next tab                  |
| `Shift+Tab`   | Previous tab              |
| `1`           | Go directly to Markets    |
| `2`           | Go directly to Portfolio  |

Tab switching is suppressed when a text input is active (creating a portfolio,
filtering coins, entering an amount). In those modes, all keystrokes go to
the input.

Global quit: `q` or `Ctrl+C` from any tab, any mode except active text input.
`Ctrl+C` always quits even from inside a text input.

---

## Tab 1 — Markets

Displays a scrollable list of coins ordered by market cap rank. Prices refresh
automatically in the background.

### Columns

| Column     | Content                                    |
|------------|--------------------------------------------|
| `#`        | Market cap rank                            |
| `Name`     | Coin name                                  |
| `Ticker`   | Symbol (uppercased)                        |
| `Price (USD)` | Current price. Format: `$X,XXX.XX` (2 decimal places, thousands separator) for prices ≥ $1; `$0.XXXXXX` (6 decimal places) for prices < $1 |
| `24h`      | 24 h price change. Format: `+X.XX%` or `-X.XX%`. Positive values are highlighted green, negative red |

### Keyboard commands

| Key       | Action                                          |
|-----------|-------------------------------------------------|
| `j` / `↓` | Move cursor down one row                        |
| `k` / `↑` | Move cursor up one row                          |
| `g`       | Jump to first coin (top of list)                |
| `G`       | Jump to last coin (bottom of list)              |
| `r`       | Manually trigger a price refresh. No-op if a refresh is already in progress |
| `q`       | Quit                                            |

The table scrolls to keep the cursor visible. The cursor row is visually
highlighted.

### Data loading

**On first launch** (empty database): fetches the top 100 coins from
`/coins/markets` and stores them all locally. The table shows a loading message
until this completes.

**On subsequent launches**: loads coins directly from the local database. No
network request on startup.

**Auto-refresh**: every 60 seconds, the app fetches updated prices from
`/simple/price` for all coins in the database and updates the table
automatically. The user does not need to do anything.

A background ticker fires every 5 seconds to check whether the 60-second
threshold has elapsed. This means the auto-refresh fires within 5 seconds of
the threshold being crossed.

### Status bar

A persistent bar at the bottom of the tab shows:

- Left: available keyboard shortcuts
- Right (one of):
  - `synced Xs ago` — time since last successful refresh
  - `refreshing...` — refresh in progress
  - `error: <message>` — last error, truncated to fit. Bar background changes
    to indicate error state. For API errors, the message includes the HTTP
    status code and the error body returned by the API (e.g.
    `error: 429 you've exceeded the rate limit`). For network errors, the
    Go error string is used directly (e.g. `error: request timeout`).
  - `loading...` — initial load not yet complete

Errors are non-fatal. The table remains usable with stale data while an error
is shown.

---

## Tab 2 — Portfolio

A two-panel view for managing portfolios and holdings.

```
┌────────────────────┬──────────────────────────────────┐
│  Portfolio list    │  Holdings for selected portfolio  │
│  (left panel)      │  (right panel)                   │
└────────────────────┴──────────────────────────────────┘
[ status bar ]
```

### Left panel — portfolio list

Shows all portfolios by name. The selected portfolio is marked with a `▶`
cursor. The panel is active when in **menu mode**; it appears dimmed when focus
moves to the right panel.

If no portfolios exist, the panel shows a prompt to create one.

### Right panel — holdings

Shows the holdings for the currently selected portfolio. Header line shows the
portfolio name and the total value of all holdings in USD.

If the portfolio has no holdings, a message prompts the user to add one.

Holdings are loaded from the local SQLite database. No loading state is shown —
the panel updates immediately on the next render after the query returns.

#### Holdings table columns

| Column   | Content                                                       |
|----------|---------------------------------------------------------------|
| `Coin`   | Coin name                                                     |
| `Ticker` | Symbol (uppercased)                                           |
| `Amount` | Quantity held, 4 decimal places (e.g. `0.5000`)               |
| `Price`  | Current price in USD (same format as Markets tab)             |
| `Value`  | `amount × price`, formatted as `$X,XXX.XX`                   |
| `24h`    | 24 h change percentage, green/red                             |
| `%`      | Proportion of this holding in the portfolio total, 1 decimal  |

Holdings are ordered by value, descending (largest position first).

### Focus modes and navigation

The portfolio tab has two navigation modes and five dialog modes.

#### Menu mode (default)

Active when no dialog is open and focus is on the left panel.

| Key             | Action                                                                 |
|-----------------|------------------------------------------------------------------------|
| `j` / `↓`      | Move to next portfolio. Immediately loads its holdings in the right panel |
| `k` / `↑`      | Move to previous portfolio. Immediately loads its holdings             |
| `PgDn` / `Ctrl+F` | Scroll the holdings preview down (half a page) without entering list mode |
| `PgUp` / `Ctrl+B` | Scroll the holdings preview up                                        |
| `Enter`         | Enter **list mode** (move focus to the right panel / holdings list). Only available when the selected portfolio has at least one holding |
| `n`             | Open **Create Portfolio** dialog                                       |
| `a`             | Open **Add Holding** flow (coin picker). Only available when at least one portfolio exists |

#### List mode

Active when focus is on the holdings list in the right panel.

| Key       | Action                                                            |
|-----------|-------------------------------------------------------------------|
| `j` / `↓` | Move cursor down one holding                                     |
| `k` / `↑` | Move cursor up one holding                                       |
| `g`       | Jump to first holding                                             |
| `G`       | Jump to last holding                                              |
| `Enter`   | Open **Edit Holding** dialog for the selected holding             |
| `X`       | Open **Delete Holding** confirmation for the selected holding     |
| `a`       | Open **Add Holding** flow (coin picker)                           |
| `Esc`     | Return to menu mode                                               |

The list scrolls to keep the cursor visible.

### Dialogs

All dialogs are centered overlays. The background panels are still rendered
behind them. Each dialog has its own status bar hint line.

---

#### Create Portfolio

Opened with `n` from menu mode.

A text input for the portfolio name (max 50 characters, placeholder: `e.g. Long Term`).

| Key     | Action                                                          |
|---------|-----------------------------------------------------------------|
| `Enter` | Create the portfolio (no-op if input is empty after trimming)   |
| `Esc`   | Cancel and return to menu mode                                  |
| Any other key | Forwarded to the text input                              |

**After creation:** the new portfolio is inserted into the database, the list
reloads, and the cursor automatically moves to the newly created portfolio. Focus
moves to list mode immediately so the user can start adding holdings.

---

#### Add Holding — step 1: Select Coin

Opened with `a` from either menu mode or list mode.

A searchable, scrollable list of all coins in the local database.

A filter input at the top (placeholder: `filter coins...`, max 30 characters).
Typing any character immediately filters the coin list. Matching is
case-insensitive substring search across coin name, ticker, and CoinGecko API
ID simultaneously.

If no coins are in the database (Markets tab has never loaded), a message
prompts the user to visit the Markets tab first.

| Key       | Action                                                             |
|-----------|--------------------------------------------------------------------|
| `j` / `↓` | Move cursor down in the filtered list                             |
| `k` / `↑` | Move cursor up in the filtered list                               |
| `Enter`   | Select the highlighted coin and proceed to step 2                 |
| `Esc`     | Cancel and return to whichever mode was active before (`a` was pressed) |
| Any other key | Forwarded to the filter input; list re-filters immediately   |

The cursor is clamped to the filtered list length as the filter changes.

---

#### Add Holding — step 2: Enter Amount

Shown after a coin is selected in step 1.

Displays the selected coin's name and ticker. A numeric input for the quantity
(placeholder: `e.g. 0.5`, max 20 characters).

| Key     | Action                                                                      |
|---------|-----------------------------------------------------------------------------|
| `Enter` | Validate and save. Rejects non-numeric input or values ≤ 0 with an inline error message. On success, adds the holding and returns to the previous mode |
| `Esc`   | Go back to step 1 (coin selection). The filter and cursor are preserved     |
| Any other key | Forwarded to the amount input                                       |

**After saving:** both the portfolio list and the holdings list reload. The
portfolio sidebar reflects any updated total value. If the coin already exists
in this portfolio, the amount is updated (not duplicated).

---

#### Edit Holding

Opened with `Enter` on a holding in list mode.

Displays the coin name, ticker, and current amount. A numeric input
pre-populated with the current amount (4 decimal places).

| Key     | Action                                                                      |
|---------|-----------------------------------------------------------------------------|
| `Enter` | Validate and save new amount. Same validation as Add Holding. Returns to list mode |
| `Esc`   | Cancel without saving. Returns to list mode                                 |
| Any other key | Forwarded to the amount input                                       |

---

#### Delete Holding

Opened with `X` on a holding in list mode.

Displays the coin name, ticker, and current amount. Asks for confirmation before
deleting.

| Key     | Action                                           |
|---------|--------------------------------------------------|
| `Enter` | Confirm deletion. Returns to list mode           |
| `Esc`   | Cancel. Returns to list mode without deleting    |

No other keys are active in this dialog (they are silently ignored).

**After deletion:** the holdings list reloads. The cursor stays at the same
position (clamped if the deleted item was the last one).

---

### Portfolio status bar

The status bar hint line changes based on the active mode:

| Mode              | Hint shown                                                              |
|-------------------|-------------------------------------------------------------------------|
| Menu mode         | `j/k portfolios • PgUp/PgDn preview • Enter navigate • n new • a add holding • Tab markets • q quit` |
| List mode         | `j/k holdings • g/G top/bottom • Enter edit • X delete • a add holding • Esc back to menu • q quit` |
| Create dialog     | `Enter to create • Esc to cancel`                                       |
| Coin picker       | `j/k navigate • type to filter • Enter select • Esc cancel`            |
| Amount input      | `Enter to confirm • Esc back to coin selection`                         |
| Edit dialog       | `Enter to save • Esc to cancel`                                         |
| Delete confirm    | `Enter to delete • Esc to cancel`                                       |

If an error is present, it is appended to the hint and the bar changes
appearance to draw attention.

---

## Data directory and storage

The database file is stored at:

```
$XDG_DATA_HOME/crypto_tracker/data.db
```

If `XDG_DATA_HOME` is not set, defaults to:

```
~/.local/share/crypto_tracker/data.db
```

The directory is created automatically on first launch. The database schema is
applied automatically on every open (using `CREATE TABLE IF NOT EXISTS`), so
there is no separate migration step.

---

## Environment variables

| Variable            | Required | Description                                         |
|---------------------|----------|-----------------------------------------------------|
| `COINGECKO_API_KEY` | No       | CoinGecko demo API key for higher rate limits       |
| `XDG_DATA_HOME`     | No       | Override for the data directory (XDG spec)          |

---

## Behaviours not yet implemented

These are features that are natural next steps but were not built in the
prototype:

- **Delete portfolio.** There is no way to remove a portfolio. Holdings would
  be cascade-deleted automatically by the database.
- **Rename portfolio.** No edit flow for the portfolio name.
- **Sorting and filtering in Markets.** The list is always ordered by market cap
  rank. No way to sort by price or 24 h change, or to search by name.
- **Pagination control.** The coin limit (top 100) is hardcoded. There is no
  way to load more coins or change the limit.
- **Multiple currencies.** All values are USD only. No currency switching.
- **Historical data / charts.** No price history, no sparklines.
- **Import / export.** No way to back up or move the database.
- **Configuration file.** The refresh interval and coin limit are hardcoded.
  There is no user-editable config.
