---
branch: feat/012-coingecko-rate-limiting
revision: 1
status: done
---

# Slice 012 — CoinGecko Rate Limiting (Revision 1)

## Smoke test + completeness audit

No findings. All scope items implemented, test coverage adequate, verification
commands satisfied.

## Implementation review

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| I1 | LOW | FIXED | Throttle has theoretical race between unlock and timer fire |
| I2 | LOW | FIXED | Inconsistent `io.ReadAll` error handling in 429 path |

**I1** `internal/api/coingecko.go:92-117`  
The `throttle()` method releases `mu` before sleeping on the timer, then re-acquires it to update `lastRequestAt`. If two goroutines call `throttle()` concurrently, both could calculate short sleep durations based on the same stale `lastRequestAt`, wake at nearly the same time, and both update `lastRequestAt` — violating the minimum interval guarantee. This is mitigated in practice by the UI-level `refreshing` flag that serialises API calls, so concurrent throttle calls won't occur in the current app. If the `HTTPClient` is ever used concurrently (e.g., from multiple tabs), the throttle could allow requests closer together than `minInterval`. Consider holding the lock through the sleep (using a condition variable) or recording the intended wake time atomically before releasing the lock.

**I2** `internal/api/coingecko.go:163` and `internal/api/coingecko.go:241`  
In the 429 handler, `io.ReadAll(resp.Body)` errors are ignored (`body, _ := io.ReadAll(resp.Body)`), while the generic non-2xx handler at lines 176 and 254 properly checks `readErr`. Slice 11 explicitly fixed ignored `io.ReadAll` errors in those other paths. For a 429 response, the body is informational (used in the error message), so ignoring the read error is low-risk — the `RateLimitError` is still returned with status 429. However, it's inconsistent with the codebase convention established in Slice 11. Consider handling the error the same way: `body, readErr := io.ReadAll(resp.Body)` and incorporating `readErr` into the `RateLimitError.Body` or `Error()` string if non-nil.