package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredericomozzato/crypto_tracker/internal/api"
	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

// settingsMode is a discriminated union for SettingsModel states.
type settingsMode interface{ isSettingsMode() }

type (
	settingsBrowsing struct{}
	settingsPicking  struct {
		filter   textinput.Model
		all      []store.Currency
		filtered []store.Currency
		cursor   int
		offset   int // viewport offset for scrolling
	}
)

func (settingsBrowsing) isSettingsMode() {}
func (settingsPicking) isSettingsMode()  {}

// SettingsModel manages the Settings tab UI.
type SettingsModel struct {
	ctx          context.Context
	store        store.Store
	client       api.CoinGeckoClient
	width        int
	height       int
	mode         settingsMode
	selectedCode string
	currencies   []store.Currency
	loading      bool
	lastErr      string
}

// Message types for SettingsModel.
type settingsLoadedMsg struct {
	currencies []store.Currency
	selected   string
}

type settingsNeedFetchMsg struct{}

type currenciesFetchedMsg struct {
	codes []string
}

type currenciesUpsertedMsg struct {
	currencies []store.Currency
}

// NewSettingsModel creates a new SettingsModel with the given dependencies.
func NewSettingsModel(ctx context.Context, s store.Store, c api.CoinGeckoClient) SettingsModel {
	return SettingsModel{
		ctx:          ctx,
		store:        s,
		client:       c,
		mode:         settingsBrowsing{},
		selectedCode: "usd",
	}
}

// Init returns the initial command to load settings data.
func (m SettingsModel) Init() tea.Cmd {
	return m.cmdLoadSettings()
}

// update is the internal update function that handles messages.
func (m SettingsModel) update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case settingsLoadedMsg:
		m.currencies = msg.currencies
		if msg.selected != "" {
			m.selectedCode = msg.selected
		}

	case settingsNeedFetchMsg:
		m.loading = true
		return m, m.cmdFetchCurrencies()

	case currenciesFetchedMsg:
		// Filter API codes against fiat map
		fiatCodes := api.FilterFiat(msg.codes)
		currencies := make([]store.Currency, len(fiatCodes))
		for i, code := range fiatCodes {
			name, _ := api.FiatCurrencyName(code)
			currencies[i] = store.Currency{
				Code: code,
				Name: name,
			}
		}
		return m, m.cmdUpsertCurrencies(currencies)

	case currenciesUpsertedMsg:
		m.currencies = msg.currencies
		m.loading = false
		// Transition to picking mode
		m.mode = m.makePickingMode(msg.currencies)

	case errMsg:
		m.loading = false
		m.lastErr = msg.err.Error()

	case tea.KeyMsg:
		switch mode := m.mode.(type) {
		case settingsBrowsing:
			switch msg.Type {
			case tea.KeyEnter:
				if len(m.currencies) == 0 {
					return m, func() tea.Msg { return settingsNeedFetchMsg{} }
				}
				m.mode = m.makePickingMode(m.currencies)
			case tea.KeyEscape:
				// Esc does nothing in browsing mode (user can use Tab/1-3 to switch tabs)
				return m, nil
			}

		case settingsPicking:
			picking := mode

			switch msg.Type {
			case tea.KeyEscape:
				m.mode = settingsBrowsing{}
				return m, nil

			case tea.KeyEnter:
				// No-op for Slice 13 - selection handled in Slice 14
				return m, nil

			case tea.KeyDown:
				if picking.cursor < len(picking.filtered)-1 {
					picking.cursor++
				}
				picking.adjustViewport()
				m.mode = picking
				return m, nil

			case tea.KeyUp:
				if picking.cursor > 0 {
					picking.cursor--
				}
				picking.adjustViewport()
				m.mode = picking
				return m, nil

			case tea.KeyRunes:
				// Forward all character keys to filter input (no j/k navigation)
				var cmd tea.Cmd
				picking.filter, cmd = picking.filter.Update(msg)
				picking.filtered = filterCurrencies(picking.all, picking.filter.Value())
				// Clamp cursor
				if picking.cursor >= len(picking.filtered) && len(picking.filtered) > 0 {
					picking.cursor = len(picking.filtered) - 1
				}
				m.mode = picking
				return m, cmd

			default:
				// Forward all other keys (Backspace, Delete, etc.) to filter input
				var cmd tea.Cmd
				picking.filter, cmd = picking.filter.Update(msg)
				picking.filtered = filterCurrencies(picking.all, picking.filter.Value())
				// Clamp cursor
				if picking.cursor >= len(picking.filtered) && len(picking.filtered) > 0 {
					picking.cursor = len(picking.filtered) - 1
				}
				m.mode = picking
				return m, cmd
			}
		}
	}

	return m, nil
}

// makePickingMode creates a picking mode with initialized filter input.
func (m SettingsModel) makePickingMode(currencies []store.Currency) settingsPicking {
	filter := textinput.New()
	filter.Placeholder = "Search currencies..."
	filter.Prompt = "" // Remove the default "> " prompt
	filter.Focus()

	return settingsPicking{
		filter:   filter,
		all:      currencies,
		filtered: currencies,
		cursor:   0,
		offset:   0,
	}
}

// filterCurrencies returns currencies matching the filter string.
func filterCurrencies(currencies []store.Currency, filter string) []store.Currency {
	if filter == "" {
		return currencies
	}
	filter = strings.ToLower(filter)
	result := make([]store.Currency, 0)
	for _, c := range currencies {
		if strings.Contains(strings.ToLower(c.Code), filter) ||
			strings.Contains(strings.ToLower(c.Name), filter) {
			result = append(result, c)
		}
	}
	return result
}

// adjustViewport updates offset so the cursor stays visible.
// The picker displays at most maxVisibleItems (10) currencies at once.
func (p *settingsPicking) adjustViewport() {
	const maxVisibleItems = 10
	visibleRows := maxVisibleItems
	if visibleRows > len(p.filtered) {
		visibleRows = len(p.filtered)
	}
	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+visibleRows {
		p.offset = p.cursor - visibleRows + 1
	}
	maxOffset := len(p.filtered) - visibleRows
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.offset > maxOffset {
		p.offset = maxOffset
	}
	if p.offset < 0 {
		p.offset = 0
	}
}

// View renders the Settings tab content.
func (m SettingsModel) View() string {
	switch mode := m.mode.(type) {
	case settingsBrowsing:
		return m.viewBrowsing()
	case settingsPicking:
		return m.viewPicking(mode)
	default:
		return m.viewBrowsing()
	}
}

func (m SettingsModel) viewBrowsing() string {
	if m.loading {
		return "Loading currencies…\n" + m.renderStatusBar()
	}

	var b strings.Builder

	// Panel title
	titleStyle := lipgloss.NewStyle().Bold(true)
	b.WriteString(titleStyle.Render("Settings") + "\n")

	// Show current currency as a selectable row
	currencyName := m.selectedCode
	if name, ok := api.FiatCurrencyName(m.selectedCode); ok {
		currencyName = name
	}
	line := fmt.Sprintf("  Base Currency: %s (%s)", strings.ToUpper(m.selectedCode), currencyName)

	// Highlight the row (cursor is always on this single row in browsing mode)
	highlight := lipgloss.NewStyle().Reverse(true)
	line = highlight.Render(line)

	b.WriteString(line + "\n")

	// Calculate panel inner height: total height minus borders (2) minus status bar (1)
	innerHeight := m.height - 3
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Pad content to fill inner height
	lines := strings.Split(b.String(), "\n")
	for len(lines) < innerHeight {
		lines = append(lines, "")
	}
	content := strings.Join(lines, "\n")

	// Render with panel border for visual framing
	accentColor := lipgloss.Color("#00FFFF")
	panelStyle := lipgloss.NewStyle().
		Width(m.width - 2).
		Height(innerHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(accentColor)

	return panelStyle.Render(content) + "\n" + m.renderStatusBar()
}

func (m SettingsModel) viewPicking(picking settingsPicking) string {
	var b strings.Builder

	// Header
	b.WriteString("Select Base Currency\n")
	b.WriteString("\n")

	// Filter input
	b.WriteString(picking.filter.View())
	b.WriteString("\n\n")

	// Currency list - show at most 10 items to keep dialog compact
	const maxVisibleItems = 10
	if len(picking.filtered) == 0 {
		b.WriteString("No currencies match your search.\n")
	} else {
		visibleRows := maxVisibleItems
		if visibleRows > len(picking.filtered) {
			visibleRows = len(picking.filtered)
		}
		end := picking.offset + visibleRows
		if end > len(picking.filtered) {
			end = len(picking.filtered)
		}

		for i := picking.offset; i < end; i++ {
			c := picking.filtered[i]
			var line string
			if i == picking.cursor {
				// Highlight selected
				selectedStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("#4A4A4A")).
					Foreground(lipgloss.Color("#FFFFFF"))
				line = selectedStyle.Render(fmt.Sprintf("> %s - %s", strings.ToUpper(c.Code), c.Name))
			} else {
				line = fmt.Sprintf("  %s - %s", strings.ToUpper(c.Code), c.Name)
			}
			b.WriteString(line + "\n")
		}
	}

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Render(b.String())

	// Place dialog in center without pushing tab bar off-screen
	// Leave one row for the status bar to prevent overflow
	content := lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, dialog)
	return content + "\n" + m.renderStatusBar()
}

// renderStatusBar returns a status bar with keyboard hints.
func (m SettingsModel) renderStatusBar() string {
	var content string
	switch m.mode.(type) {
	case settingsBrowsing:
		content = "Enter to change • Tab/1-3 switch tabs • q quit"
	case settingsPicking:
		content = "↑/↓ navigate • type to filter • Esc cancel"
	}

	if m.lastErr != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
		content += " • " + errStyle.Render("error: "+m.lastErr)
	}

	return content
}

// InputActive returns true when the currency picker is open.
func (m SettingsModel) InputActive() bool {
	_, ok := m.mode.(settingsPicking)
	return ok
}

// cmdLoadSettings creates a command to load settings from the database.
func (m SettingsModel) cmdLoadSettings() tea.Cmd {
	return func() tea.Msg {
		currencies, err := m.store.GetAllCurrencies(m.ctx)
		if err != nil {
			return errMsg{err: err}
		}
		selected, err := m.store.GetSetting(m.ctx, "selected_currency")
		if err != nil {
			return errMsg{err: err}
		}
		return settingsLoadedMsg{
			currencies: currencies,
			selected:   selected,
		}
	}
}

// cmdFetchCurrencies creates a command to fetch supported currencies from the API.
func (m SettingsModel) cmdFetchCurrencies() tea.Cmd {
	return func() tea.Msg {
		codes, err := m.client.FetchSupportedCurrencies(m.ctx)
		if err != nil {
			return errMsg{err: err}
		}
		return currenciesFetchedMsg{codes: codes}
	}
}

// cmdUpsertCurrencies creates a command to persist currencies to the database.
func (m SettingsModel) cmdUpsertCurrencies(currencies []store.Currency) tea.Cmd {
	return func() tea.Msg {
		if err := m.store.UpsertCurrencies(m.ctx, currencies); err != nil {
			return errMsg{err: err}
		}
		return currenciesUpsertedMsg{currencies: currencies}
	}
}
