---
status: in_review
branch: feat/011-code-correctness-clarity
---

# Slice 11 — Code correctness & clarity

Small, targeted code changes with no new runtime behaviour.

## Context

Slices 1–10 are DONE. The app has a working markets tab, portfolio tab with full CRUD, auto-refresh, and status bar. This slice fixes bugs, cleans up inconsistencies, adds documentation comments, and names magic numbers.

## Scope

1. Fix ignored `io.ReadAll` errors in `internal/api/coingecko.go:86, 144`
2. Clean up unused `database` local in `cmd/crypto-tracker/main.go:42` — add lifecycle-ownership comment
3. Named constants for magic numbers: 100-coin fetch limit and 60s refresh threshold in `internal/ui/markets.go`
4. Add comment documenting quit-time cancellation assumption in `cmd/crypto-tracker/main.go:32-33, 66`
5. Add comment explaining implicit message broadcast pattern in `internal/ui/app.go:102-108`

## Data model

No schema changes.

## Files to modify

### 1. `internal/api/coingecko.go`

**Fix silently dropped `io.ReadAll` errors in non-2xx branches.**

Lines 86 and 144 both do `body, _ := io.ReadAll(resp.Body)` — the `_` silently discards potential read errors. If reading the body fails, the status code is reported but the body content is empty or truncated, producing a misleading error message.

**Change in `FetchMarkets`:** Replace lines ~85-88:

```go
// Before:
if resp.StatusCode < 200 || resp.StatusCode >= 300 {
    body, _ := io.ReadAll(resp.Body)
    return nil, fmt.Errorf("fetching markets: %d %s", resp.StatusCode, string(body))
}

// After:
if resp.StatusCode < 200 || resp.StatusCode >= 300 {
    body, readErr := io.ReadAll(resp.Body)
    if readErr != nil {
        return nil, fmt.Errorf("fetching markets: %d (failed to read response body: %w)", resp.StatusCode, readErr)
    }
    return nil, fmt.Errorf("fetching markets: %d %s", resp.StatusCode, string(body))
}
```

**Change in `FetchPrices`:** Same pattern at lines ~143-146:

```go
// Before:
if resp.StatusCode < 200 || resp.StatusCode >= 300 {
    body, _ := io.ReadAll(resp.Body)
    return nil, fmt.Errorf("fetching prices: %d %s", resp.StatusCode, string(body))
}

// After:
if resp.StatusCode < 200 || resp.StatusCode >= 300 {
    body, readErr := io.ReadAll(resp.Body)
    if readErr != nil {
        return nil, fmt.Errorf("fetching prices: %d (failed to read response body: %w)", resp.StatusCode, readErr)
    }
    return nil, fmt.Errorf("fetching prices: %d %s", resp.StatusCode, string(body))
}
```

### 2. `internal/ui/markets.go`

**Replace magic numbers with named constants.**

Add at package level (near `staleThreshold`):

```go
const (
    coinFetchLimit   = 100
    refreshInterval  = 60 * time.Second
    staleThreshold   = 5 * time.Minute // already exists, include in group
)
```

Remove the existing standalone `staleThreshold` const since it moves into this block.

**Update `cmdLoad`:** Line 81 (`len(existing) >= 100` → `len(existing) >= coinFetchLimit`) and line 85 (`m.client.FetchMarkets(m.ctx, 100)` → `m.client.FetchMarkets(m.ctx, coinFetchLimit)`).

**Update tick handler:** Line 140 (`time.Since(m.lastRefreshed) >= 60*time.Second` → `time.Since(m.lastRefreshed) >= refreshInterval`).

### 3. `cmd/crypto-tracker/main.go`

**Add two documentation comments.**

**Comment 1 — Cancellation safety:** Above `ctx, cancel := signal.NotifyContext(...)` (lines 32-33), add:

```go
// Root context is cancelled on SIGINT/SIGTERM, propagating cancellation to
// all in-flight HTTP requests and DB queries. This is safe because every DB
// write is an atomic upsert — there is no risk of partial or corrupt data on
// abrupt shutdown. Must revisit if multi-statement transactions are added.
```

**Comment 2 — Lifecycle ownership:** Above `database, err := db.Open(...)` (line 42), add:

```go
// The raw *sql.DB handle is passed to the store and not used directly after
// this point. The store owns the DB lifecycle (including Close via defer below).
```

### 4. `internal/ui/app.go`

**Add comment explaining broadcast pattern.** Above the fan-out block (lines ~102-108), add:

```go
// Background messages (non-key, non-resize) are always forwarded to both
// children via tea.Batch. This ensures that async responses like
// coinsLoadedMsg and pricesUpdatedMsg reach whichever tab issued the
// command, even if the user has since switched tabs. Without this
// broadcast, responses would be silently dropped when the inactive tab
// doesn't match the issuing tab.
```

## Tests

No new tests required. Verify existing suite stays green via `make test`.

## Implementation order

1. `internal/api/coingecko.go` — Fix `io.ReadAll` error handling
2. `internal/ui/markets.go` — Named constants for magic numbers
3. `cmd/crypto-tracker/main.go` — Documentation comments
4. `internal/ui/app.go` — Broadcast-pattern comment
5. `make check` — verify all tests pass, lint clean, no vet errors

## Verification

```bash
make fmt
make lint
make test
make vuln
make build
```

All must pass. No new test files. No new runtime behaviour.

## Branch name

`feat/011-code-correctness-clarity`