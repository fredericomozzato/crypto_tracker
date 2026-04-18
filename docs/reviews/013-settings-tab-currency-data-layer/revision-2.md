---
branch: feat/013-settings-tab-currency-data-layer
revision: 2
status: done
---

# Slice 013 — Settings tab + currency data layer (Revision 2)

## Smoke test + completeness audit

`make check` passes (fmt, lint, test, vuln all green). Binary compiles cleanly.

Scope item gaps from user feedback:

- **Settings browsing mode has no title and doesn't follow the panel layout convention.** The Markets tab renders a table with column headers and highlighted row. The Portfolio tab renders a two-panel bordered layout with titled panels ("Portfolios", "Holdings"). The Settings browsing mode renders "Base Currency: USD (US Dollar)" as plain unstyled text inside a single bordered panel — no section title, no row highlight, no consistent visual language.

- **Browsing mode has no cursor/highlight on the current setting row.** The issue spec describes settings as a selectable list (cursor on "Base Currency" row, Enter opens picker). Currently `settingsBrowsing` renders just text — there is no indication which row is selected or that any row is interactive.

- **Picking mode j/k conflict with filter typing.** Pressing `j` or `k` in picking mode navigates the cursor instead of being typed into the filter. The user reports this prevents searching for currencies like "JPY" (Japanese Yen). User requests removing j/k navigation entirely and supporting only Up/Down arrow keys. Also, the `textinput.Prompt` defaults to `"> "`, which appears as a `>` prefix in the search field — the user finds this confusing and wants it removed.

## Implementation review

| ID  | Sev  | Status | Summary |
|-----|------|--------|---------|
| F1  | HIGH | FIXED | Settings browsing mode has no title, no row highlight, no panel convention consistency |
| F2  | HIGH | FIXED | Picking mode: j/k cannot be typed into filter — user requests arrow-only navigation |
| F3  | MED  | FIXED | Picking mode: `>` prompt symbol in textinput is confusing; remove it |
| I1  | HIGH | FIXED | Viewport calculation mismatch: `adjustViewport` uses `height-4` but `viewPicking` renders `height-8` visible rows |
| I2  | MED  | FIXED | Settings model doesn't handle `errMsg` — async errors from `cmdLoadSettings`, `cmdFetchCurrencies`, `cmdUpsertCurrencies` are silently discarded |
| I3  | LOW  | FIXED | Selected-row rendering uses `line[2:]` slicing instead of building the line with selection indicator |

**F1** `internal/ui/settings.go:272-303`
The `viewBrowsing` method renders plain text ("Base Currency: USD (US Dollar)") inside a bordered panel. It lacks: (a) a panel title consistent with other tabs (e.g. "Settings" header), (b) a highlighted/selectable row for the "Base Currency" setting item, (c) any visual indication that the row is interactive or selected. The Markets tab uses `highlight.Render(line)` for the cursor row. The Portfolio tab uses `▶` prefix and accent-colored borders for focused panels. The Settings tab should follow the same convention — render each setting as a selectable row with cursor highlight, so that when additional settings are added in future slices, the pattern is already established.

**F2** `internal/ui/settings.go:156-174`
When in `settingsPicking` mode, pressing `j` or `k` is intercepted for cursor movement and the key returns early — it never reaches the filter `textinput.Model`. This makes it impossible to search for currencies whose code contains "j" or "k" (e.g. JPY, KRW, KZT). The user's requested resolution is to remove j/k navigation entirely and only support `KeyUp`/`KeyDown` arrows for cursor movement in the currency picker, freeing j/k to be typed into the filter.

**F3** `internal/ui/settings.go:207`
`makePickingMode` creates `textinput.New()` without setting `filter.Prompt = ""`. The default `Prompt` in `bubbles/textinput` is `"> "`, which renders a `>` prefix in the search field. The user finds this confusing (it looks like a cursor indicator, not a prompt). Set `filter.Prompt = ""` to remove it.

**I1** `internal/ui/settings.go:237-258` vs `305-326`
`adjustViewport` calculates `visibleRows = height - 4` (reserving 4 lines for header, filter, viewport calculations), while `viewPicking` renders `visibleRows = m.height - 8` (reserving 8 lines for header, filter, dialog border, status). These different overhead calculations mean the scroll offset computed by `adjustViewport` will be out of sync with what's actually visible, potentially scrolling the cursor out of view.

**I2** `internal/ui/settings.go:79-202`
The `update` method handles `settingsLoadedMsg`, `settingsNeedFetchMsg`, `currenciesFetchedMsg`, and `currenciesUpsertedMsg`, but has **no case for `errMsg`**. The `cmdLoadSettings`, `cmdFetchCurrencies`, and `cmdUpsertCurrencies` commands all return `errMsg` on failure. Without handling this message type, errors are silently discarded — the `m.lastErr` field is never set and the user sees no feedback. Every other tab (Markets, Portfolio) has an `errMsg` handler that sets `lastErr` to the error string.

**I3** `internal/ui/settings.go:332-340`
The selected row in `viewPicking` is rendered by building each line as `"  %s - %s"`, then replacing the selected row with `"> " + line[2:]`. The `line[2:]` slicing assumes every line starts with exactly 2 characters and is fragile. A better approach is to build the line conditionally:

```go
if i == picking.cursor {
    line = selectedStyle.Render(fmt.Sprintf("%s - %s", strings.ToUpper(c.Code), c.Name))
} else {
    line = fmt.Sprintf("  %s - %s", strings.ToUpper(c.Code), c.Name)
}
```