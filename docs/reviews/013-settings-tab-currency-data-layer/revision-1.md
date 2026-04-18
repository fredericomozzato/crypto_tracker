---
branch: feat/013-settings-tab-currency-data-layer
revision: 1
status: done
---

# Slice 013 — Settings tab + currency data layer (Revision 1)

## Smoke test + completeness audit

All tests pass (`make check` green), binary compiles cleanly, no panics on launch.

Scope item gaps:

- **Arrow keys (Up/Down) not handled in picking mode.** The issue spec says `"j"/"↓" move cursor down` and `"k"/"↑" move cursor up`. The Markets and Portfolio coin-picker both handle `tea.KeyDown`/`tea.KeyUp`. The Settings picker only handles `j`/`k` via `tea.KeyRunes` — arrow keys are silently dropped.

- **Esc/q handling in browsing mode is incomplete.** The issue spec says: `"Esc"/"q" → returns to tab bar`. The browsing mode in `settings.go` has no key handler for `Esc` or `q` at all. These keys work at the `AppModel` level (`q` quits globally, `Esc` is not handled for tab switching), but `Esc` in browsing mode does not return focus to the tab bar as specified.

- **Status bar hint line is missing.** Every other tab/mode in the app has a dedicated status bar at the bottom of the view with keyboard hints. The Settings tab renders hints as inline text within the content area.

## Implementation review

| ID  | Sev  | Status | Summary |
|-----|------|--------|---------|
| F1  | BLOCKER | FIXED | Picking mode drops Backspace/Delete/arrow keys — no `default` case in key handler |
| F2  | BLOCKER | FIXED | Picking mode has no viewport/scroll management — list renders all items, appears scrolled to bottom |
| F3  | HIGH  | FIXED | j/k consumed for navigation, cannot be typed into the filter |
| F4  | HIGH  | FIXED | Settings view has no visual framing or layout — inconsistent with rest of the app |
| F5  | HIGH  | FIXED | Instructions rendered inline instead of in a status bar at the bottom |
| F6  | MED   | FIXED | Arrow keys (Up/Down) not handled in picking mode |
| F7  | MED   | FIXED | No Esc handling in browsing mode to return to tab bar |
| I1  | HIGH  | FIXED | cmdLoadSettings/cmdFetchCurrencies/cmdUpsertCurrencies use `context.Background()` instead of the model's `ctx` |
| I2  | LOW   | FIXED | `currencies` table uses `code TEXT NOT NULL UNIQUE` instead of `code TEXT PRIMARY KEY` (issue spec says `code TEXT PK`) |
| I3  | LOW   | FIXED | `FilterFiat` result is unsorted when currencies come from the API path — only DB-retrieved currencies are sorted by code |

**F1** `internal/ui/settings.go:125-161`
The `settingsPicking` key handler has cases for `KeyEscape`, `KeyEnter`, and `KeyRunes` only. There is no `default` case. Keys like `KeyBackspace`, `KeyDelete`, and `KeyDown`/`KeyUp` fall through with no action and no forwarding to `textinput.Model.Update()`. The user cannot delete characters from the filter input. The portfolio's `addCoin` mode has the correct pattern: a `default` case that delegates to `mode.filter.Update(msg)`.

**F2** `internal/ui/settings.go:236-267` (`viewPicking`)
The picker renders every currency in `picking.filtered` with no offset or viewport calculation. With ~35 fiat currencies, the list overflows the terminal and the rendered output starts from the first item — but since the cursor starts at item 0 and the terminal scrolls to fit all content, the visible area shows the bottom of the list. The Markets tab uses `m.offset`/`m.adjustViewport()` to keep the cursor visible; the Settings picker has no equivalent.

**F3** `internal/ui/settings.go:134-150`
When the user is in picking mode and presses `j` or `k`, the key is intercepted for cursor navigation and returns early. The user cannot type `j` or `k` into the filter. This differs from the portfolio coin picker where the same behavior exists (j/k navigate), but in that context the coin picker's filter uses `placeholder: "filter coins..."` and the expectation is search-by-name/ticker. For the currency picker, the same trade-off applies but the user reported being unable to filter by code containing 'j' or 'k'. The code is working as designed for navigation, but the design means you cannot search for currencies like `jpy` (Japanese Yen) by typing their code.

**F4** `internal/ui/settings.go:210-233` (`viewBrowsing`) and `236-267` (`viewPicking`)
The browsing mode renders plain unstyled lines with no panel border, no visual hierarchy, and no framing. The picking mode renders a plain list with no dialog border or centering. Every dialog in the portfolio tab uses `lipgloss.Place()` with `lipgloss.RoundedBorder()` and `lipgloss.Center` positioning. The settings tab should follow the same conventions.

**F5** `internal/ui/settings.go:224-225` and `265-266`
The browsing mode appends inline text (`"Press Enter to change currency"`, `"Press Tab to switch tabs, q to quit"`). The picking mode appends a one-liner at the bottom of the list content. Neither uses a dedicated status bar row. The Markets and Portfolio tabs both have a `renderStatusBar()` method that renders hints at the bottom of the viewport using `lipgloss.Width()`-based alignment, consistent with the app's convention.

**F6** `internal/ui/settings.go:125-161`
The issue spec says `j`/`↓` move down and `k`/`↑` move up. Only `j`/`k` (via `KeyRunes`) are handled. `tea.KeyDown` and `tea.KeyUp` are not handled in picking mode. The Markets tab and portfolio coin-picker both handle arrow keys.

**F7** `internal/ui/settings.go:114-120`
The `settingsBrowsing` key handler only processes `KeyEnter`. There is no handler for `KeyEscape`. The issue spec says `Esc` in browsing mode should return to the tab bar. Currently pressing Esc in browsing mode does nothing.

**I1** `internal/ui/settings.go:277-316`
`cmdLoadSettings`, `cmdFetchCurrencies`, and `cmdUpsertCurrencies` all create `context.Background()` inside the command closure instead of capturing the model's `m.ctx`. Per architecture conventions, the root context (cancelled on quit) should be used so that in-flight I/O is cancelled when the user quits. The Markets model correctly uses `m.ctx` in its command closures.

**I2** `internal/db/schema.sql:26-29`
The issue spec defines `code TEXT PK` but the implemented schema is `code TEXT NOT NULL UNIQUE`. While functionally equivalent in SQLite, `PRIMARY KEY` is the convention used by all other tables and is what the spec requests.

**I3** `internal/api/fiat.go:45-52` + `internal/ui/settings.go:94-104`
`FilterFiat` returns codes in the order they appear in the API response, which is not guaranteed to be sorted. When currencies are loaded from the database (via `GetAllCurrencies`), they are sorted by code (`ORDER BY code ASC`). But when loaded fresh from the API (via `currenciesFetchedMsg`), they are in whatever order the API returns. This means the list order can change between sessions.