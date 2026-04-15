---
status: in_review
branch: feat/012-coingecko-rate-limiting
---

# Slice 12 — CoinGecko Rate Limiting

New runtime behaviour: the app must stay within the free-tier limit of ~30 req/min.

## Context

Slices 1–11 are DONE. The app has a working Markets tab with auto-refresh (every 60s), manual refresh (`r` key), and status bar states (Synced, Stale, Refreshing, error, loading). The `HTTPClient` makes two types of requests: `FetchMarkets` on initial seed and `FetchPrices` on refreshes. A `refreshing` mutex flag prevents concurrent requests at the UI level. However, there is no protection against CoinGecko's ~30 req/min demo-tier rate limit, no 429 error handling, and no backoff strategy.

## Scope

From the roadmap:

1. Implement request throttling in `internal/api/coingecko.go`
2. Detect 429 responses and back off when rate-limited
3. Extend the `r` no-op to also suppress when rate-limited
4. Surface rate-limit status in the status bar (e.g. `Rate limited — retrying in Xs`)
5. TDD: throttle logic, backoff behaviour, status bar state for rate-limited condition

## Data model

No SQL schema changes. New application-level types only.

## Files to create/modify

### 1. `internal/api/coingecko.go` — Add throttling + 429 detection

**New type:**

```go
// RateLimitError is returned when the CoinGecko API responds with HTTP 429.
type RateLimitError struct {
    Body       string
    RetryAfter time.Duration // cooldown duration; 0 means use a default
}

func (e *RateLimitError) Error() string {
    return fmt.Sprintf("rate limited: 429 %s", e.Body)
}
```

**New helper functions:**

```go
// IsRateLimitError reports whether err is a *RateLimitError.
func IsRateLimitError(err error) bool {
    var rle *RateLimitError
    return errors.As(err, &rle)
}

// RetryAfterFromError extracts the RetryAfter from a RateLimitError.
// Returns defaultDuration if err is not a RateLimitError or has RetryAfter == 0.
func RetryAfterFromError(err error, defaultDuration time.Duration) time.Duration {
    var rle *RateLimitError
    if errors.As(err, &rle) && rle.RetryAfter > 0 {
        return rle.RetryAfter
    }
    return defaultDuration
}
```

**New constant:**

```go
const DefaultRetryAfter = 60 * time.Second
```

**Changes to `HTTPClient`:**

Add fields for throttling:

```go
type HTTPClient struct {
    httpClient    *http.Client
    baseURL       string
    apiKey        string
    mu            sync.Mutex
    lastRequestAt time.Time
    minInterval   time.Duration // minimum gap between API calls (default 2s)
}
```

- `NewHTTPClient` initializes `minInterval` to `2 * time.Second` and sets `lastRequestAt` to zero time.
- Add a `throttle(ctx context.Context) error` method that sleeps until `minInterval` has elapsed since `lastRequestAt`, then updates `lastRequestAt`. Respects context cancellation.
- Both `FetchMarkets` and `FetchPrices` call `c.throttle(ctx)` as the first action before building the request.

**429 detection in `FetchMarkets` and `FetchPrices`:**

Add an explicit 429 handler before the generic non-2xx handler:

```go
if resp.StatusCode == http.StatusTooManyRequests {
    body, _ := io.ReadAll(resp.Body)
    rle := &RateLimitError{
        Body: string(body),
    }
    if retryAfterHeader := resp.Header.Get("Retry-After"); retryAfterHeader != "" {
        if seconds, err := strconv.Atoi(retryAfterHeader); err == nil {
            rle.RetryAfter = time.Duration(seconds) * time.Second
        }
    }
    return nil, rle
}
```

**New unexported constructor for testing:**

```go
// newHTTPClientWithInterval creates an HTTPClient with a custom minimum request interval.
// Used for testing; production code should use NewHTTPClient.
func newHTTPClientWithInterval(apiKey string, interval time.Duration, serverURL string) *HTTPClient {
    // ...
}
```

### 2. `internal/api/coingecko_test.go` — Tests for throttling + 429

**New tests:**

1. **`TestThrottleEnforcesMinimumInterval`** — Calls `FetchMarkets` twice in quick succession on an `httptest.NewServer`. Verify the second call takes at least `minInterval` to return.

2. **`TestThrottleWithContextCancellation`** — Cancels context during throttle wait. Verify `FetchMarkets` returns context.Canceled error promptly.

3. **`TestFetchMarketsRateLimit429`** — httptest server returns 429 with body `"rate limit exceeded"`. Verify that `FetchMarkets` returns a `*RateLimitError` with `Body` containing the response text.

4. **`TestFetchPricesRateLimit429`** — Same as above for `FetchPrices`.

5. **`TestFetchMarkets429WithRetryAfterHeader`** — httptest server returns 429 with `Retry-After: 30` header. Verify `RateLimitError.RetryAfter == 30s`.

6. **`TestIsRateLimitError`** — Unit test for the helper with both `*RateLimitError` and non-rate-limit errors.

7. **`TestRetryAfterFromError`** — Unit test: returns error's `RetryAfter` if set, else returns default. Also test with non-`RateLimitError`.

8. **`TestNewHTTPClientDefaultMinInterval`** — Verify `NewHTTPClient("")` sets `minInterval` to 2 seconds.

### 3. `internal/ui/markets.go` — Rate-limit state + status bar

**New fields on `MarketsModel`:**

```go
rateLimitedUntil time.Time // time at which rate-limit cooldown expires
refreshAttempts  int       // consecutive rate-limit errors (for backoff)
```

**Modified `errMsg` handler in `update()`:**

```go
case errMsg:
    m.refreshing = false
    if api.IsRateLimitError(msg.err) {
        backoff := api.RetryAfterFromError(msg.err, api.DefaultRetryAfter)
        multiplier := 1 << min(m.refreshAttempts, 3) // 1, 2, 4, 8 (capped)
        cooldown := backoff * time.Duration(multiplier)
        if cooldown > 5*time.Minute {
            cooldown = 5 * time.Minute
        }
        m.rateLimitedUntil = time.Now().Add(cooldown)
        m.refreshAttempts++
        m.lastErr = "" // status bar shows rate-limit state, not raw error
    } else {
        m.lastErr = msg.err.Error()
    }
```

**Modified `r` key handler:**

```go
case 'r':
    if m.refreshing || time.Now().Before(m.rateLimitedUntil) || len(m.coins) == 0 {
        return m, nil // no-op: already refreshing, rate-limited, or no data
    }
    m.refreshing = true
    return m, m.cmdRefresh()
```

**Modified `tickMsg` handler:**

```go
case tickMsg:
    cmds := []tea.Cmd{cmdTick()}
    now := time.Now()
    canRefresh := !m.refreshing && len(m.coins) > 0 && now.Sub(m.lastRefreshed) >= refreshInterval && now.After(m.rateLimitedUntil)
    if canRefresh {
        m.refreshing = true
        cmds = append(cmds, m.cmdRefresh())
    }
    return m, tea.Batch(cmds...)
```

**Modified `pricesUpdatedMsg` handler:**

```go
case pricesUpdatedMsg:
    m.coins = msg.coins
    m.refreshing = false
    m.lastErr = ""
    m.lastRefreshed = time.Now()
    m.refreshAttempts = 0           // reset backoff on success
    m.rateLimitedUntil = time.Time{} // clear rate-limit state
```

**Modified `statusRight()`:**

```go
func (m MarketsModel) statusRight() string {
    now := time.Now()
    if m.refreshing {
        return "Refreshing"
    }
    if now.Before(m.rateLimitedUntil) {
        secs := int(time.Until(m.rateLimitedUntil).Seconds())
        return fmt.Sprintf("Rate limited — retry in %ds", secs)
    }
    if m.lastErr != "" {
        return "error: " + m.lastErr
    }
    if m.lastRefreshed.IsZero() {
        return "loading..."
    }
    if now.Sub(m.lastRefreshed) > staleThreshold {
        return "Stale"
    }
    return "Synced"
}
```

**Modified `renderStatusBar()`:**

Change the status bar switch from exact string match to `strings.HasPrefix` for the rate-limit case:

```go
var rightStyled string
switch {
case strings.HasPrefix(rightContent, "Rate limited"):
    rightStyled = rateLimitStyle.Render(rightContent)
case rightContent == "Synced":
    rightStyled = greenStyle.Render(rightContent)
case rightContent == "Stale":
    rightStyled = yellowStyle.Render(rightContent)
case strings.HasPrefix(rightContent, "error:"):
    rightStyled = errStyle.Render(rightContent)
default:
    rightStyled = grayStyle.Render(rightContent)
}
```

New style constant in `renderStatusBar`:

```go
rateLimitStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C00")) // dark orange
```

### 4. `internal/ui/markets_test.go` — Tests for rate-limit state

**New tests:**

1. **`TestMarketsRateLimitedErrMsg`** — Send `errMsg{err: &api.RateLimitError{...}}` to model. Verify `rateLimitedUntil` is set to future time and `refreshAttempts` is 1.

2. **`TestMarketsRateLimitedStatusBar`** — Set `rateLimitedUntil` to future time. Verify `statusRight()` returns `"Rate limited — retry in Xs"` with correct seconds.

3. **`TestMarketsRateLimitedStatusBarNoLongerLimited`** — Set `rateLimitedUntil` to past time. Verify `statusRight()` returns normal state.

4. **`TestMarketsRefreshBlockedWhenRateLimited`** — Set `rateLimitedUntil` to future time, send `r` key. Verify no refresh command returned and `refreshing` stays false.

5. **`TestMarketsAutoRefreshBlockedWhenRateLimited`** — Set `rateLimitedUntil` to future time, send `tickMsg` with elapsed interval. Verify no refresh command.

6. **`TestMarketsAutoRefreshResumesAfterCooldown`** — Set `rateLimitedUntil` to past time, send `tickMsg` with elapsed interval. Verify refresh fires.

7. **`TestMarketsExponentialBackoffOnRepeated429`** — Send multiple `errMsg` with `RateLimitError`. Verify `rateLimitedUntil` increases exponentially (60s → 120s → 240s → 480s → capped at 300s).

8. **`TestMarketsBackoffResetOnSuccess`** — Send `errMsg` with `RateLimitError`, then `pricesUpdatedMsg`. Verify `refreshAttempts` resets to 0 and `rateLimitedUntil` is zero time.

9. **`TestMarketsNonRateLimitErrorDoesNotSetRateLimitedUntil`** — Send `errMsg{err: errors.New("network failed")}`. Verify `rateLimitedUntil` stays zero.

10. **`TestMarketsRateLimitedStatusBarStyled`** — Verify the rendered status bar contains the rate-limit text.

### 5. `internal/ui/testhelpers_test.go` — No structural changes needed

`StubAPI.err` already supports returning arbitrary errors including `*api.RateLimitError`.

## Implementation order

1. Define `RateLimitError`, `IsRateLimitError`, `RetryAfterFromError`, and `DefaultRetryAfter` in `internal/api/coingecko.go`
2. Add throttle fields and `throttle()` method to `HTTPClient`
3. Add 429 detection in `FetchMarkets` and `FetchPrices`
4. Write tests for `RateLimitError`, throttle, and 429 detection in `internal/api/coingecko_test.go`
5. Add `rateLimitedUntil` and `refreshAttempts` fields to `MarketsModel`
6. Modify `errMsg` handler in `update()` to detect `RateLimitError` and set backoff
7. Modify `r` key handler to no-op when rate-limited
8. Modify `tickMsg` handler to respect rate-limit cooldown
9. Modify `pricesUpdatedMsg` handler to reset backoff state
10. Modify `statusRight()` to show rate-limited countdown
11. Modify `renderStatusBar()` to style the rate-limited state
12. Write tests for all MarketsModel rate-limit state transitions in `internal/ui/markets_test.go`
13. Run `make check` and fix any issues

## Verification

```bash
make fmt     # gofumpt formatting
make lint    # golangci-lint
make test    # go test -race -coverprofile=coverage.out ./...
make vuln    # govulncheck
make check   # all of the above
```

All tests must pass non-interactively with no network access. The throttle tests use short intervals (e.g. 10ms) to keep test execution fast.

## Branch

`feat/012-coingecko-rate-limiting`