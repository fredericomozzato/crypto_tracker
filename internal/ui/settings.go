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
	return cmdLoadSettings(m.store)
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
		return m, cmdFetchCurrencies(m.client)

	case currenciesFetchedMsg:
		// Filter API codes against fiat map
		fiatCodes := api.FilterFiat(msg.codes)
		currencies := make([]store.Currency, len(fiatCodes))
		for i, code := range fiatCodes {
			currencies[i] = store.Currency{
				Code: code,
				Name: api.FiatCurrencies[code],
			}
		}
		return m, cmdUpsertCurrencies(m.store, currencies)

	case currenciesUpsertedMsg:
		m.currencies = msg.currencies
		m.loading = false
		// Transition to picking mode
		m.mode = m.makePickingMode(msg.currencies)

	case tea.KeyMsg:
		switch mode := m.mode.(type) {
		case settingsBrowsing:
			if msg.Type == tea.KeyEnter {
				if len(m.currencies) == 0 {
					return m, func() tea.Msg { return settingsNeedFetchMsg{} }
				}
				m.mode = m.makePickingMode(m.currencies)
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

			case tea.KeyRunes:
				for _, r := range msg.Runes {
					switch r {
					case 'j', 'J':
						if picking.cursor < len(picking.filtered)-1 {
							picking.cursor++
						}
						m.mode = picking
						return m, nil
					case 'k', 'K':
						if picking.cursor > 0 {
							picking.cursor--
						}
						m.mode = picking
						return m, nil
					}
				}
				// Forward to filter input
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
	filter.Focus()

	return settingsPicking{
		filter:   filter,
		all:      currencies,
		filtered: currencies,
		cursor:   0,
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
		return "Loading currencies…"
	}

	var b strings.Builder

	// Show current currency
	currencyName := m.selectedCode
	if name, ok := api.FiatCurrencies[m.selectedCode]; ok {
		currencyName = name
	}
	_, _ = fmt.Fprintf(&b, "Base Currency: %s (%s)\n", strings.ToUpper(m.selectedCode), currencyName)
	b.WriteString("\n")
	b.WriteString("Press Enter to change currency\n")
	b.WriteString("Press Tab to switch tabs, q to quit\n")

	if m.lastErr != "" {
		b.WriteString("\n")
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
		b.WriteString(errStyle.Render("Error: " + m.lastErr))
	}

	return b.String()
}

func (m SettingsModel) viewPicking(picking settingsPicking) string {
	var b strings.Builder

	// Header
	b.WriteString("Select Base Currency\n")
	b.WriteString("\n")

	// Filter input
	b.WriteString(picking.filter.View())
	b.WriteString("\n\n")

	// Currency list
	if len(picking.filtered) == 0 {
		b.WriteString("No currencies match your search.\n")
	} else {
		for i, c := range picking.filtered {
			line := fmt.Sprintf("  %s - %s", strings.ToUpper(c.Code), c.Name)
			if i == picking.cursor {
				// Highlight selected
				selectedStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("#4A4A4A")).
					Foreground(lipgloss.Color("#FFFFFF"))
				line = selectedStyle.Render("> " + line[2:])
			}
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString("j/k to navigate, type to filter, Enter to select (disabled), Esc to cancel\n")

	return b.String()
}

// InputActive returns true when the currency picker is open.
func (m SettingsModel) InputActive() bool {
	_, ok := m.mode.(settingsPicking)
	return ok
}

// cmdLoadSettings creates a command to load settings from the database.
func cmdLoadSettings(s store.Store) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		currencies, err := s.GetAllCurrencies(ctx)
		if err != nil {
			return errMsg{err: err}
		}
		selected, err := s.GetSetting(ctx, "selected_currency")
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
func cmdFetchCurrencies(client api.CoinGeckoClient) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		codes, err := client.FetchSupportedCurrencies(ctx)
		if err != nil {
			return errMsg{err: err}
		}
		return currenciesFetchedMsg{codes: codes}
	}
}

// cmdUpsertCurrencies creates a command to persist currencies to the database.
func cmdUpsertCurrencies(s store.Store, currencies []store.Currency) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := s.UpsertCurrencies(ctx, currencies); err != nil {
			return errMsg{err: err}
		}
		return currenciesUpsertedMsg{currencies: currencies}
	}
}
