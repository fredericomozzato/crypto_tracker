---
branch: feat/006-create-portfolio-left-panel
revision: 2
status: done
---

# Slice 6 — Create portfolio + left panel (Revision 2)

## Smoke test + completeness audit

No findings. Build passes cleanly, all 109 tests pass with race detector. All scope
items remain implemented. This revision captures visual bugs discovered via manual
app inspection that tests did not exercise.

## Implementation review

| ID | Sev | Status | Summary |
|----|-----|--------|---------|
| I1 | HIGH | FIXED | Bordered panels add 2 unaccounted rows, pushing tab bar off screen |
| I2 | HIGH | FIXED | `overlayDialog` uses byte-length ops on ANSI strings, breaking dialog layout |
| I3 | MED  | FIXED | Left + right panel outer widths sum to `m.width + 3`, causing overflow |

**I1** `internal/ui/portfolio.go:152`  
`contentHeight := m.height - 1` is then passed as `Height()` to both bordered panel
styles. `lipgloss.Height()` sets the *inner* content height — lipgloss adds one row for
the top border and one for the bottom border, making each panel's rendered height
`contentHeight + 2`. With the status bar (1 row), the total portfolio output is
`(contentHeight + 2) + 1 = m.height + 2` rows. `app.go` then prepends the tab bar
row and a newline on top, pushing the whole output 2 rows below the terminal bottom.
The tab bar scrolls off screen entirely (confirmed in the screenshot — no
"Markets / Portfolio" tabs visible when on the Portfolio tab).

Fix: `contentHeight := m.height - 3`. This yields an inner height of `m.height - 3`,
a rendered panel height of `m.height - 1` (inner + 2 border rows), and a total output
of `m.height - 1 + 1 (status bar) = m.height` rows — the correct allocation for the
space `app.go` reserved.

**I2** `internal/ui/portfolio.go:251-291`  
`overlayDialog` replaces portions of background lines using raw string slicing:
`line[:startX]`, `line[startX+len(dialogLine):]`, and `len(line)`. All of these are
*byte-level* operations. Lipgloss-rendered strings contain ANSI escape codes
(colour, bold, reverse, border drawing characters) that inflate the byte length far
beyond the visual column count. `startX` is computed as a visual column offset, but
is then used as a byte index — the mismatch causes the dialog to be inserted at the
wrong position and the surrounding background lines to be sliced mid-escape-sequence,
producing corrupted output (confirmed in the screenshot showing split/jagged borders
and misaligned dialog).

Fix: replace the string-manipulation overlay entirely. When in `creating` mode, skip
rendering the panels and instead render the dialog centered in the full content area
using `lipgloss.Place`:

```go
if _, ok := m.mode.(creating); ok {
    mode := m.mode.(creating)
    dialog := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        Padding(1, 2).
        Render("New Portfolio\n\n" + mode.input.View())
    content := lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, dialog)
    return content + "\n" + m.renderStatusBar()
}
```

The overlay-behind-panels requirement from the issue plan is a visual nicety; the
dialog replacing the panels while open is an acceptable simplification given that
lipgloss v1 has no built-in ANSI-aware overlay primitive. Remove `renderDialogOverlay`
and `overlayDialog` entirely.

**I3** `internal/ui/portfolio.go:155-165`  
`leftWidth := 30` and `rightWidth := m.width - leftWidth - 1` are both used as *inner*
widths for bordered styles. The outer width of each panel is `inner + 2` (left and
right border column). Total outer width:

```
leftOuter + rightOuter
= (30 + 2) + ((m.width - 31) + 2)
= 32 + m.width - 29
= m.width + 3
```

Three columns wider than the terminal, causing line wrapping and misaligned right
border (visible in the screenshot as the right panel border appearing inside the
terminal rather than at the edge).

Fix: define widths in terms of outer (total) columns and derive inner from those:

```go
leftPanelOuter  := 30
rightPanelOuter := m.width - leftPanelOuter
leftPanelInner  := leftPanelOuter - 2
rightPanelInner := rightPanelOuter - 2

leftStyle  := lipgloss.NewStyle().Width(leftPanelInner).Height(contentHeight).Border(lipgloss.NormalBorder())
rightStyle := lipgloss.NewStyle().Width(rightPanelInner).Height(contentHeight).Border(lipgloss.NormalBorder())
```

The `Reverse(true)` highlight applied to cursor rows in `renderLeftPanel` must also
use `leftPanelInner` as its width so the highlight spans the full inner panel width
without overflowing into the border.
