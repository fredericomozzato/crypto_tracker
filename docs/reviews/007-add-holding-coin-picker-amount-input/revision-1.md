---
branch: feat/007-add-holding-coin-picker-amount-input
revision: 1
status: done
---

# Slice 007 — Add Holding: Coin Picker + Amount Input (Revision 1)

## Smoke test + completeness audit

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| F1 | LOW | FIXED | Two specified tests not implemented: `TestCoinPickerTypingFilters` and `TestCoinPickerCursorClampedAfterFilter` |

**F1** `internal/ui/portfolio_test.go`  
The issue spec lists `TestCoinPickerTypingFilters` (typing a character reduces `mode.filtered` to matching coins only) and `TestCoinPickerCursorClampedAfterFilter` (cursor at 2 before a filter yielding 1 result → cursor clamped to 0) as required tests. Neither is present. The underlying behavior is implemented in code but has no test coverage — the clamp bug in I1 below was not caught as a result.

---

## Implementation review

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| I1 | HIGH | FIXED | Cursor goes negative when `j` pressed with empty filtered list; subsequent Enter panics with index out of bounds |
| I2 | LOW | FIXED | Filter and amount inputs are missing placeholder text and `CharLimit` per PRD |

**I1** `internal/ui/portfolio.go:173,187-189,596-599`  
When the filter query matches no coins, `len(mode.filtered) == 0`. Pressing `j` executes:
```go
mode.cursor = intMin(mode.cursor+1, len(mode.filtered)-1)
// = intMin(1, -1) = -1
```
The upper-bound clamp at lines 187–189 only fires when `cursor >= len(filtered)`. For cursor=-1, `-1 >= 0` is false, so the cursor stays at -1. If the user then clears the filter (making coins visible again) and presses Enter, `transitionToAddAmount` at line 599 executes `mode.filtered[-1]` → runtime panic.

Fix: change the clamp to also handle the lower bound:
```go
if len(mode.filtered) == 0 {
    mode.cursor = 0
} else if mode.cursor >= len(mode.filtered) {
    mode.cursor = len(mode.filtered) - 1
} else if mode.cursor < 0 {
    mode.cursor = 0
}
```
Or simply guard the `j` case: `if len(mode.filtered) > 0 { mode.cursor = ... }`.

**I2** `internal/ui/portfolio.go:276-279,601`  
The PRD specifies:
- Coin picker filter: placeholder `"filter coins..."`, max 30 characters
- Amount input: placeholder `"e.g. 0.5"`, max 20 characters

Current code uses `ti.Placeholder = "search..."` for the filter and `ti.Placeholder = "amount"` for the amount input. Neither sets `ti.CharLimit`. Apply the PRD-specified values when constructing both text inputs.
