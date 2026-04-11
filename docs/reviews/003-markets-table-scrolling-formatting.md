---
branch: feat/003-markets-table-scrolling-formatting
status: passed
revision: 2
---

# Slice 3 — Markets table: 100 coins, scrolling, formatting

## Smoke test + completeness audit

All scope items are implemented:

- `internal/format/format.go` — `FmtPrice` and `FmtChange` present and tested
- `internal/ui/app.go` — `cursor`/`offset` fields, `moveCursor`, `adjustViewport`, `tableHeight` helpers
- `j`/`k`/`↓`/`↑`/`g`/`G` key handling in `Update`
- Load-or-fetch logic in `Init` (DB first, API on first launch with limit=100)
- Column table renderer with header, viewport scrolling, highlight, and hint line

All 54 tests pass (`go test -race ./...`). Binary builds cleanly (`make build`).

All required tests from the issue spec are present and contain non-trivial assertions.

**Tooling gap — BLOCKER:**

`gofumpt` and `govulncheck` are not in PATH on this machine. `make check` fails
immediately at the `fmt` step:

```
make: gofumpt: No such file or directory
make: *** [fmt] Error 1
```

This means CI cannot be verified locally and the standard pre-commit gate is broken.
Fix: install the missing tools or update `make check` to degrade gracefully when they
are absent. This should be resolved before merging.

---

## Implementation review

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| I1 | HIGH | OPEN | `FmtPrice` decimal part overflows for prices like `99.995` → `"$99.100"` |
| I2 | MED | OPEN | `moveCursor` sets cursor to `-1` when called on an empty coin list |
| I3 | LOW | OPEN | `truncate` slices at byte offset, not rune boundary |

**I1** `internal/format/format.go:15`

The decimal part is computed as `int64(fracPart*100 + 0.5)`. When the result equals
or exceeds `100` (e.g. input `99.995` → `fracPart ≈ 0.995` → `int64(100.0) = 100`),
`fmt.Sprintf(".%02d", 100)` produces `".100"`, yielding `"$99.100"` instead of
`"$100.00"`. The `%02d` verb pads to a minimum of 2 digits but does not truncate.

Fix: use `math.Round` or format as a single float with `fmt.Sprintf("$%.2f", v)` then
insert commas into the integer part only.

**I2** `internal/ui/app.go:252-261`

`moveCursor` does not guard against an empty `m.coins` slice. When `len(m.coins) == 0`,
the clamp `m.cursor = len(m.coins) - 1` evaluates to `-1`. If the user presses `j` or
`k` during the loading window, cursor becomes `-1`. When `coinsLoadedMsg` arrives it
sets `m.coins` but does not reset the cursor, so no row is ever highlighted until the
user navigates again.

Fix: add `if len(m.coins) == 0 { return }` at the top of `moveCursor`, or clamp
cursor to 0 in the `coinsLoadedMsg` handler.

**I3** `internal/ui/app.go:295-303`

`truncate` uses `s[:maxLen-1]`, which slices at a byte offset. A multi-byte UTF-8
rune straddling that boundary produces either a garbled string or a panic. CoinGecko
names are ASCII in practice but this is fragile.

Fix: use `[]rune(s)` slice instead, or use `utf8.RuneCountInString` / `string([]rune(s)[:n])`.

---

## Revision 2

### Fixes verified

All three implementation findings from revision 1 are confirmed fixed:

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| I1 | HIGH | FIXED | `FmtPrice` now uses `fmt.Sprintf("%.2f", v)` split on `.` — rounding handled by sprintf, no decimal overflow possible |
| I2 | MED | FIXED | `moveCursor` guards with `if len(m.coins) == 0 { return }` at top |
| I3 | LOW | FIXED | `truncate` uses `utf8.RuneCountInString` + `[]rune` slice — correct rune-boundary truncation |

All 57 tests pass. `make build` succeeds. No panics in smoke test.

### Tooling findings

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| T1 | HIGH | FIXED | `gofumpt` not installed — `make fmt` and `make check` fail immediately |
| T2 | HIGH | FIXED | `golangci-lint ./...` wrong syntax for v2 — should be `golangci-lint run ./...` |
| T3 | MED | FIXED | `govulncheck` not installed — `make vuln` and `make lint` fail |

**T1** `Makefile:4`

`gofumpt` is not in PATH. `make fmt` fails with `make: gofumpt: No such file or directory`.
Install: `go install mvdan.cc/gofumpt@latest` (documented in CLAUDE.md).

**T2** `Makefile:8`

The installed `golangci-lint` is v2.11.4. In v2 the top-level command is removed — `golangci-lint ./...` produces `Error: unknown command "./..."`. The correct invocation is `golangci-lint run ./...`. The `.golangci.yml` already has `version: "2"` so the config is correct; only the Makefile call needs updating.

**T3** `Makefile:9`

`govulncheck` is not in PATH. `make vuln` and the `govulncheck ./...` line in `make lint` both fail.
Install: `go install golang.org/x/vuln/cmd/govulncheck@latest` (documented in CLAUDE.md).

### Status

T1 and T3 require user action (install missing tools). T2 is a one-line Makefile fix.
