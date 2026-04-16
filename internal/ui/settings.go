package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

// settingsMode is the discriminated union for settings tab modes.
type settingsMode interface{ isSettingsMode() }

type (
	settingsBrowsing        struct{}
	settingsPickingCurrency struct {
		filter   textinput.Model
		filtered []string
		cursor   int
		offset   int
	}
)

func (settingsBrowsing) isSettingsMode()        {}
func (settingsPickingCurrency) isSettingsMode() {}

// currenciesLoadedMsg is sent when currencies and the current setting are loaded from the DB.
type currenciesLoadedMsg struct {
	codes   []string
	current string
}

// SettingsModel manages the Settings tab.
type SettingsModel struct {
	ctx        context.Context
	store      store.Store
	currencies []string // full list loaded from DB
	current    string   // active currency code
	mode       settingsMode
	width      int
	height     int
}

// NewSettingsModel creates a new SettingsModel with the given dependencies.
func NewSettingsModel(ctx context.Context, s store.Store) SettingsModel {
	return SettingsModel{
		ctx:     ctx,
		store:   s,
		current: "usd",
		mode:    settingsBrowsing{},
	}
}

// Init loads currencies and the current setting from the DB.
func (m SettingsModel) Init() tea.Cmd {
	return m.cmdLoadCurrencies()
}

func (m SettingsModel) cmdLoadCurrencies() tea.Cmd {
	return func() tea.Msg {
		codes, err := m.store.GetCachedCurrencies(m.ctx)
		if err != nil {
			return errMsg{err: err}
		}
		current, _ := m.store.GetSetting(m.ctx, "currency")
		if current == "" {
			current = "usd"
		}
		return currenciesLoadedMsg{codes: codes, current: current}
	}
}

func newCurrencyFilterInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.Focus()
	return ti
}

// pickerVisibleHeight returns the number of currency rows visible in the picker dialog.
func (m SettingsModel) pickerVisibleHeight() int {
	// Overhead: border (2) + padding (2×2) + title (1) + blank (1) + filter (1) + blank (1) = 10
	h := m.height - 10
	if h < 3 {
		h = 3
	}
	if h > 15 {
		h = 15
	}
	return h
}

// cursorForCurrent returns the index of m.current in m.currencies (0 if not found).
func (m SettingsModel) cursorForCurrent() int {
	for i, code := range m.currencies {
		if code == m.current {
			return i
		}
	}
	return 0
}

// update handles all messages for the settings tab.
func (m SettingsModel) update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case currenciesLoadedMsg:
		m.currencies = msg.codes
		m.current = msg.current
		return m, nil

	case tea.KeyMsg:
		switch mode := m.mode.(type) {
		case settingsBrowsing:
			switch msg.Type {
			case tea.KeyEnter:
				if len(m.currencies) == 0 {
					return m, nil
				}
				cursor := m.cursorForCurrent()
				offset := 0
				h := m.pickerVisibleHeight()
				if cursor >= h {
					offset = cursor - h + 1
				}
				m.mode = settingsPickingCurrency{
					filter:   newCurrencyFilterInput(),
					filtered: m.currencies,
					cursor:   cursor,
					offset:   offset,
				}
				return m, nil
			}

		case settingsPickingCurrency:
			switch msg.Type {
			case tea.KeyEsc:
				m.mode = settingsBrowsing{}
				return m, nil
			case tea.KeyEnter:
				// No-op in PR 1 — just close the picker without changing anything
				m.mode = settingsBrowsing{}
				return m, nil
			case tea.KeyUp:
				mode.cursor = intMax(mode.cursor-1, 0)
				if mode.cursor < mode.offset {
					mode.offset = mode.cursor
				}
				m.mode = mode
				return m, nil
			case tea.KeyDown:
				if mode.cursor < len(mode.filtered)-1 {
					mode.cursor++
					h := m.pickerVisibleHeight()
					if mode.cursor >= mode.offset+h {
						mode.offset = mode.cursor - h + 1
					}
				}
				m.mode = mode
				return m, nil
			case tea.KeyRunes:
				for _, r := range msg.Runes {
					switch r {
					case 'j', 'J':
						if mode.cursor < len(mode.filtered)-1 {
							mode.cursor++
							h := m.pickerVisibleHeight()
							if mode.cursor >= mode.offset+h {
								mode.offset = mode.cursor - h + 1
							}
						}
						m.mode = mode
						return m, nil
					case 'k', 'K':
						mode.cursor = intMax(mode.cursor-1, 0)
						if mode.cursor < mode.offset {
							mode.offset = mode.cursor
						}
						m.mode = mode
						return m, nil
					}
				}
				// Any other rune goes to the filter input
				newInput, cmd := mode.filter.Update(msg)
				mode.filter = newInput
				mode.filtered = filterCurrencies(m.currencies, mode.filter.Value())
				mode.cursor = 0
				mode.offset = 0
				m.mode = mode
				return m, cmd
			default:
				// Backspace, Ctrl+A, etc. — delegate to filter
				newInput, cmd := mode.filter.Update(msg)
				mode.filter = newInput
				mode.filtered = filterCurrencies(m.currencies, mode.filter.Value())
				if mode.cursor >= len(mode.filtered) {
					mode.cursor = intMax(len(mode.filtered)-1, 0)
				}
				if mode.cursor < mode.offset {
					mode.offset = mode.cursor
				}
				m.mode = mode
				return m, cmd
			}
		}
	}

	return m, nil
}

// InputActive returns true when the currency picker is open.
// This suppresses tab switching while the user navigates or types in the picker.
func (m SettingsModel) InputActive() bool {
	_, ok := m.mode.(settingsPickingCurrency)
	return ok
}

// View renders the settings page.
func (m SettingsModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	switch mode := m.mode.(type) {
	case settingsPickingCurrency:
		dialog := m.renderCurrencyPicker(mode)
		content := lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, dialog)
		return content + "\n" + m.renderStatusBar()
	}

	return m.renderBrowsing() + "\n" + m.renderStatusBar()
}

func (m SettingsModel) renderBrowsing() string {
	titleStyle := lipgloss.NewStyle().Bold(true)
	grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA"))
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#2A2A2A")).
		Padding(0, 1)
	rowStyle := lipgloss.NewStyle().Reverse(true)

	label := "Base Currency"
	value := strings.ToUpper(m.current)

	// Build the row: label on left, value box on right
	const labelWidth = 24
	padded := fmt.Sprintf("%-*s", labelWidth, label)
	row := "  " + padded + valueStyle.Render(value)
	row = rowStyle.Render(row)

	hint := grayStyle.Render("(no currencies loaded — try restarting)")
	if len(m.currencies) > 0 {
		hint = grayStyle.Render("Enter to change")
	}

	lines := []string{
		titleStyle.Render("Settings"),
		"",
		row,
		"  " + hint,
	}

	// Pad to fill available height (minus status bar)
	for len(lines) < m.height-2 {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func (m SettingsModel) renderCurrencyPicker(mode settingsPickingCurrency) string {
	h := m.pickerVisibleHeight()
	end := mode.offset + h
	if end > len(mode.filtered) {
		end = len(mode.filtered)
	}

	var b strings.Builder
	b.WriteString("Base Currency\n\n")
	b.WriteString(mode.filter.View() + "\n\n")

	if len(mode.filtered) == 0 {
		b.WriteString("  no matches\n")
	} else {
		for i := mode.offset; i < end; i++ {
			code := mode.filtered[i]

			cursor := "  "
			if i == mode.cursor {
				cursor = "> "
			}

			marker := "  "
			if code == m.current {
				marker = "●"
			}

			_, _ = fmt.Fprintf(&b, "%s%s %s\n", cursor, marker, strings.ToUpper(code))
		}

		remaining := len(mode.filtered) - end
		if remaining > 0 {
			_, _ = fmt.Fprintf(&b, "  … %d more\n", remaining)
		}
	}

	dialogWidth := intMax(m.width/3, 30)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(dialogWidth).
		Render(b.String())
}

// renderStatusBar renders the settings tab status bar.
func (m SettingsModel) renderStatusBar() string {
	grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	var left string
	switch m.mode.(type) {
	case settingsBrowsing:
		left = "Enter open • Tab/1/2/3 navigate tabs • q quit"
	case settingsPickingCurrency:
		left = "j/k navigate • type to filter • Enter/Esc close"
	}

	count := ""
	if len(m.currencies) > 0 {
		count = fmt.Sprintf("%d currencies", len(m.currencies))
	}

	padding := m.width - lipgloss.Width(left) - lipgloss.Width(count)
	if padding < 1 {
		padding = 1
	}

	return grayStyle.Render(left) + strings.Repeat(" ", padding) + grayStyle.Render(count)
}

// filterCurrencies returns currency codes containing the query string (case-insensitive).
func filterCurrencies(currencies []string, query string) []string {
	if query == "" {
		return currencies
	}
	q := strings.ToLower(query)
	result := make([]string, 0)
	for _, code := range currencies {
		if strings.Contains(code, q) {
			result = append(result, code)
		}
	}
	return result
}
