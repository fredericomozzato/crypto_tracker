---
branch: feat/007-add-holding-coin-picker-amount-input
revision: 3
status: done
---

# Slice 007 — Add Holding: Coin Picker + Amount Input (Revision 3)

## Smoke test + completeness audit

No findings. All scope items implemented, test coverage adequate, verification
commands satisfied.

- `a` opens coin picker with filterable coin list ✓
- Select coin → amount input → upsert holding ✓
- Right panel shows holdings table (Coin, Ticker, Amount, Price, Value, 24h, %) ✓
- Holdings ordered by value descending, portfolio total in header ✓
- Duplicate prevention: held coins filtered from picker; all-held case shows error ✓
- Store tests: 10 new holding tests pass (upsert, delete, query, ordering, proportion) ✓
- Format tests: 3 new FmtMoney tests pass ✓
- UI tests: all new portfolio/picker/filter tests pass ✓

## Implementation review

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| I1 | MED | FIXED | `m.lastErr` not cleared when coin picker opens successfully |

**I1** `internal/ui/portfolio.go:300-316`

When `coinPickerReadyMsg` successfully transitions to `addCoin` mode, `m.lastErr` is
not cleared. Any error set by a previous operation (e.g. "no coins loaded — visit
Markets tab first" from a prior attempt before the Markets tab was visited) persists
and appears in the coin picker's status bar alongside the navigation hint:

```
j/k navigate • type to filter • Enter select • Esc cancel • error: no coins loaded — visit Markets tab first
```

This is an observable bug within slice 7's scope: user visits Portfolio tab before
Markets tab → gets "no coins loaded" error → visits Markets tab to load coins →
returns → presses `a` → picker opens but stale error still shows.

Fix: add `m.lastErr = ""` immediately before setting `m.mode = addCoin{...}` in
the `coinPickerReadyMsg` success path.
