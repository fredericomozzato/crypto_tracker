# Code Review: Issue 002 — One Real Coin, Full Pipeline

**Branch:** `feat/002-one-real-coin-full-pipeline`
**Spec:** `docs/issues/002-one-real-coin-full-pipeline.md`
**Against:** `CLAUDE.md`, `docs/PRD.md`, `docs/ARCHITECTURE.md`

---

## BUG-01: Wrong CoinGecko endpoint URL for `/simple/price`

- **File:** `internal/api/coingecko.go:124`
- **Severity:** High (runtime failure)
- **Detail:** URL path is `/simple/prices?` (with trailing `s`). The CoinGecko endpoint is `/simple/price` (no `s`).
- **Fix:** Change `"/simple/prices?"` to `"/simple/price?"`.

---

## BUG-02: `Init()` and `cmdRefresh()` use `context.Background()` instead of the program context

- **File:** `internal/ui/app.go:50`, `internal/ui/app.go:112`
- **Severity:** High (spec violation, broken shutdown behaviour)
- **Detail:** Both `Init()` and `cmdRefresh()` create `context.Background()` inside the `tea.Cmd` closure. This means pressing `q` or `Ctrl+C` does not cancel in-flight HTTP requests or DB queries. The CLAUDE.md and issue spec both require threading the root context through all I/O.
- **Fix:** Accept `ctx context.Context` in `NewAppModel`, store it as a field, and use it in all `tea.Cmd` closures.

---

## BUG-03: `coinsLoadedMsg` does not clear the error

- **File:** `internal/ui/app.go:97`
- **Severity:** Medium
- **Detail:** The issue spec says `coinsLoadedMsg` should "store the coins, clear error". The handler sets `m.coins` but does not set `m.errMsg = ""`. If an error occurred before coins load successfully, the error persists in the view and coin data is never rendered (because `View()` checks `m.errMsg != ""` first).
- **Fix:** Add `m.errMsg = ""` in the `coinsLoadedMsg` handler.

---

## BUG-04: `pricesUpdatedMsg` does not clear the error

- **File:** `internal/ui/app.go:99`
- **Severity:** Medium
- **Detail:** Same issue as BUG-03. The spec says `pricesUpdatedMsg` should "store the updated coins, clear refreshing and error". The handler does not clear `m.errMsg`.
- **Fix:** Add `m.errMsg = ""` in the `pricesUpdatedMsg` handler.

---

## SPEC-01: Minimum terminal size is wrong

- **File:** `internal/ui/app.go:144`
- **Severity:** Medium (spec deviation)
- **Detail:** `View()` checks `m.width < 80 || m.height < 24`. The PRD and CLAUDE.md both specify 100 columns × 30 rows as the minimum. The message text also reads `"80×24"` instead of `"100×30"`.
- **Fix:** Change condition to `m.width < 100 || m.height < 30` and update the message string.

---

## SPEC-02: `go.mod` declares `go 1.25.0` — should be `1.24`

- **File:** `go.mod:3`
- **Severity:** Medium (spec deviation, toolchain compatibility)
- **Detail:** CLAUDE.md specifies Go version 1.24. `go 1.25.0` does not exist yet and will cause issues with standard Go toolchains.
- **Fix:** Change to `go 1.24`.

---

## SPEC-03: `modernc.org/sqlite` is classified as an indirect dependency

- **File:** `go.mod:35`
- **Severity:** Low (correctness)
- **Detail:** `modernc.org/sqlite` is imported directly by `internal/db/db.go` (blank import `_ "modernc.org/sqlite"`). It should appear in the direct `require` block, not as `// indirect`.
- **Fix:** Run `go mod tidy` to fix the classification.

---

## QUALITY-01: Duplicate `StubStore` definition across test packages

- **Files:** `internal/api/coingecko_test.go:198-220`, `internal/ui/app_test.go:14-36`
- **Severity:** Low (code quality)
- **Detail:** `StubStore` is defined identically in both test files. They live in different packages so there is no compilation error, but the duplication will drift over time.
- **Fix:** Accept the duplication for now (test code) or extract to `internal/testutil`. Low priority for this slice.

---

## QUALITY-02: `StubStore` in `coingecko_test.go` is unused

- **File:** `internal/api/coingecko_test.go:198-220`
- **Severity:** Low (dead code)
- **Detail:** `StubStore` is defined in the API test file but never used by any test there. API tests use `httptest.NewServer` correctly and do not need a store stub.
- **Fix:** Remove `StubStore` from `internal/api/coingecko_test.go`.

---

## QUALITY-03: `errMsg` field name shadows the `errMsg` type

- **File:** `internal/ui/app.go:21-22` (type), `internal/ui/app.go:20` (field)
- **Severity:** Low (readability)
- **Detail:** `AppModel` has a field `errMsg string` and there is a message type also named `errMsg`. Go scoping handles it, but reading `Update` is confusing when both appear in the same block.
- **Fix:** Rename the struct field to `lastErr` or the message type to `errorMsg`.

---

## QUALITY-04: `db.Open` performs I/O without accepting `context.Context`

- **File:** `internal/db/db.go:19`
- **Severity:** Low (convention violation)
- **Detail:** CLAUDE.md requires `ctx context.Context` as the first parameter of every function that performs I/O. `db.Open` runs SQLite pragmas and schema execution via `ExecContext` but constructs its own `context.Background()` internally.
- **Fix:** Change signature to `Open(ctx context.Context, path string) (*sql.DB, error)` and pass `ctx` to the internal `ExecContext` calls.

---

## QUALITY-05: `price_change` goes stale after refreshes (known gap, not actionable)

- **File:** `internal/store/sqlite.go:93-97`
- **Severity:** Info
- **Detail:** `UpdatePrices` only updates `rate` and `updated_at`. This is correct given that `/simple/price` returns no change percentage data. However, `price_change` will become stale after the first refresh cycle. Nothing to fix in this slice — document as a known limitation to address when Slice 4 adds the auto-refresh ticker.

---

## Summary

| ID | Severity | File | Action |
|---|---|---|---|
| BUG-01 | High | `internal/api/coingecko.go:124` | Fix URL: `/simple/prices?` → `/simple/price?` |
| BUG-02 | High | `internal/ui/app.go:50,112` | Thread root context through `Init()` and `cmdRefresh()` |
| BUG-03 | Medium | `internal/ui/app.go:97` | Clear `errMsg` in `coinsLoadedMsg` handler |
| BUG-04 | Medium | `internal/ui/app.go:99` | Clear `errMsg` in `pricesUpdatedMsg` handler |
| SPEC-01 | Medium | `internal/ui/app.go:144` | Fix min terminal size to 100×30 |
| SPEC-02 | Medium | `go.mod:3` | Fix Go version to `1.24` |
| SPEC-03 | Low | `go.mod:35` | Run `go mod tidy` |
| QUALITY-01 | Low | both `*_test.go` files | Accept or extract shared test stubs |
| QUALITY-02 | Low | `internal/api/coingecko_test.go` | Remove unused `StubStore` |
| QUALITY-03 | Low | `internal/ui/app.go` | Rename field or type to avoid shadowing |
| QUALITY-04 | Low | `internal/db/db.go` | Add `context.Context` param to `Open` |
| QUALITY-05 | Info | `internal/store/sqlite.go` | Known gap — `price_change` stales after refresh |
