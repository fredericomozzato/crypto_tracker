---
branch: feat/013-settings-tab-currency-data-layer
revision: 3
status: fixed
---

# Slice 013 — Settings tab + currency data layer (Revision 3)

## Smoke test + completeness audit

`make check` passes (fmt, lint, test, vuln all green). Binary compiles cleanly.

Scope item gaps:

- **`Esc` in browsing mode is advertised but non-functional.** The status bar in browsing mode reads `"Esc back to tabs"`, implying Esc returns focus to the tab bar. The `settingsBrowsing` key handler does catch `KeyEscape` but returns `m, nil` — a no-op. Neither `SettingsModel` nor `AppModel` takes any action. The hint is a lie to the user; it should either be removed or wired to observable behavior (e.g. switch to the first tab).

## Implementation review

| ID  | Sev  | Status | Summary |
|-----|------|--------|---------|
| F1  | HIGH | FIXED  | `viewPicking` passes `m.height` to `lipgloss.Place` then appends status bar, overflowing the viewport |
| F2  | MED  | FIXED  | Status bar hint `"Esc back to tabs"` in browsing mode is shown but Esc is a no-op |
| F3  | MED  | FIXED  | Error is displayed twice in browsing mode — once inside the panel and again in the status bar |
| I1  | LOW  | FIXED  | `StubStore.supportedCurrencies` field is never read by any `StubStore` method |
| I2  | LOW  | FIXED  | `FiatCurrencies` is an exported mutable map — external packages can corrupt it |
| I3  | LOW  | FIXED  | `adjustViewport` parameter `_ int` is explicitly discarded; the function signature is stale |

**F1** `internal/ui/settings.go:362-363`
`viewPicking` ends with:
```go
content := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
return content + "\n" + m.renderStatusBar()
```
`lipgloss.Place` fills a canvas of exactly `m.height` rows. Appending `"\n" + renderStatusBar()` adds 2 further rows, overflowing the terminal viewport by 2 in alt-screen mode — the status bar is clipped and never visible. `viewBrowsing` handles this correctly: it uses `innerHeight = m.height - 3`, the border adds 2 lines, giving `m.height - 1` total before the status bar. The fix is to pass `m.height - 1` to `lipgloss.Place` so one row is left for the status bar.

**F2** `internal/ui/settings.go:120-123` and `371`
`settingsBrowsing` catches `KeyEscape` and returns `m, nil`:
```go
case tea.KeyEscape:
    // Return focus to tab bar - this is handled by AppModel
    return m, nil
```
The comment is incorrect — `AppModel` does not handle `Esc` for tab switching either. The browsing mode status bar (`line 371`) tells the user `"Esc back to tabs"` but pressing Esc does nothing observable. Either remove the hint from the status bar or implement the behavior (e.g. emit a `tea.Msg` that `AppModel` responds to by switching to `tabMarkets`). Leaving a described shortcut as a no-op breaks user trust.

**F3** `internal/ui/settings.go:284-287` and `376-378`
When `m.lastErr != ""`, the error is rendered in two separate places simultaneously. In `viewBrowsing` (lines 284–287), it is written into the panel content as `"Error: <msg>"` in red. Then `renderStatusBar` (lines 376–378) appends it again to the status bar as `"error: <msg>"`. A user who hits an async error sees the same message twice on screen. The error should appear in one place only; the status bar is the consistent location across all other tabs.

**I1** `internal/ui/testhelpers_test.go:18`
`StubStore` has a `supportedCurrencies []string` field (line 18) that is set in `TestSettingsEnterTriggersFetchWhenNoCurrencies` but never read by any `StubStore` method. `FetchSupportedCurrencies` belongs to `StubAPI` (line 153), not `StubStore`. The field is dead weight that creates confusion about which stub is responsible for currency fetching. Remove it.

**I2** `internal/api/fiat.go:8`
`var FiatCurrencies = map[string]string{...}` is exported and mutable. Any package can add, remove, or overwrite entries at runtime; `settings.go` reads it by key (`api.FiatCurrencies[code]`) to display currency names. A mutation would silently produce empty or wrong names with no compile-time or runtime guard. Make the map unexported (`fiatCurrencies`) — `FilterFiat` is the only required exported entry point. If `settings.go` needs name lookup, add an unexported helper `fiatCurrencyName(code string) string` in the same file.

**I3** `internal/ui/settings.go:224`
```go
func (p *settingsPicking) adjustViewport(_ int) {
```
The `int` parameter was used in an earlier revision when viewport height was computed from terminal height. It was replaced by the hardcoded `maxVisibleItems = 10` constant (revision 2, I1 fix) but the parameter was left in the signature rather than removed. The two call sites pass `m.height` which is silently discarded. Remove the parameter and update the call sites (`lines 148` and `156`) accordingly.
