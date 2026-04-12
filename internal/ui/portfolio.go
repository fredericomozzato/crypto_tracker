package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredericomozzato/crypto_tracker/internal/format"
	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

// portfolioMode is the discriminated union for portfolio modes.
type portfolioMode interface{ isPortfolioMode() }

type (
	browsing struct{}
	creating struct{ input textinput.Model }
	addCoin  struct {
		filter   textinput.Model
		allCoins []store.Coin // already filtered — held coins removed
		filtered []store.Coin // subset matching current filter query
		cursor   int
	}
	addAmount struct {
		coin     store.Coin
		input    textinput.Model
		errMsg   string
		coinMode addCoin // preserved so Esc returns to coin picker with state intact
	}
)

func (browsing) isPortfolioMode()  {}
func (creating) isPortfolioMode()  {}
func (addCoin) isPortfolioMode()   {}
func (addAmount) isPortfolioMode() {}

// portfoliosLoadedMsg is sent when portfolios are loaded from the store.
// focusID is non-zero when the cursor should be positioned on a specific portfolio.
type portfoliosLoadedMsg struct {
	portfolios []store.Portfolio
	focusID    int64
}

// coinPickerReadyMsg is sent when all coins are loaded for the picker.
type coinPickerReadyMsg struct {
	coins []store.Coin
}

// holdingsLoadedMsg is sent when holdings are loaded for the current portfolio.
type holdingsLoadedMsg struct {
	holdings []store.HoldingRow
}

// holdingsSavedMsg is sent after a holding is successfully saved.
type holdingsSavedMsg struct {
	holdings []store.HoldingRow
}

// PortfolioModel is the Portfolio tab with two-panel layout.
type PortfolioModel struct {
	ctx        context.Context
	store      store.Store
	width      int
	height     int
	portfolios []store.Portfolio
	cursor     int
	holdings   []store.HoldingRow
	mode       portfolioMode
	lastErr    string
}

// NewPortfolioModel creates a new PortfolioModel with the given dependencies.
func NewPortfolioModel(ctx context.Context, s store.Store) PortfolioModel {
	return PortfolioModel{
		ctx:   ctx,
		store: s,
		mode:  browsing{},
	}
}

// Init loads portfolios on startup.
func (m PortfolioModel) Init() tea.Cmd {
	return m.cmdLoadPortfolios()
}

// update handles all messages and returns the updated model and command.
func (m PortfolioModel) update(msg tea.Msg) (PortfolioModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch mode := m.mode.(type) {
		case browsing:
			switch msg.Type {
			case tea.KeyRunes:
				for _, r := range msg.Runes {
					switch r {
					case 'j', 'J':
						m.moveCursor(1)
						if len(m.portfolios) > 0 {
							return m, m.cmdLoadHoldings(m.portfolios[m.cursor].ID)
						}
						return m, nil
					case 'k', 'K':
						m.moveCursor(-1)
						if len(m.portfolios) > 0 {
							return m, m.cmdLoadHoldings(m.portfolios[m.cursor].ID)
						}
						return m, nil
					case 'n', 'N':
						return m.openCreateDialog(), nil
					case 'a', 'A':
						if len(m.portfolios) > 0 {
							return m, m.cmdOpenCoinPicker()
						}
						return m, nil
					}
				}
			case tea.KeyDown:
				m.moveCursor(1)
				if len(m.portfolios) > 0 {
					return m, m.cmdLoadHoldings(m.portfolios[m.cursor].ID)
				}
				return m, nil
			case tea.KeyUp:
				m.moveCursor(-1)
				if len(m.portfolios) > 0 {
					return m, m.cmdLoadHoldings(m.portfolios[m.cursor].ID)
				}
				return m, nil
			}

		case creating:
			switch msg.Type {
			case tea.KeyEsc:
				m.mode = browsing{}
				return m, nil
			case tea.KeyEnter:
				name := strings.TrimSpace(mode.input.Value())
				if name != "" {
					return m, m.cmdCreatePortfolio(name)
				}
				return m, nil
			default:
				// Delegate to input
				newInput, cmd := mode.input.Update(msg)
				mode.input = newInput
				m.mode = mode
				return m, cmd
			}

		case addCoin:
			switch msg.Type {
			case tea.KeyEsc:
				m.mode = browsing{}
				return m, nil
			case tea.KeyEnter:
				if len(mode.filtered) > 0 {
					return m.transitionToAddAmount(mode)
				}
				return m, nil
			case tea.KeyRunes:
				for _, r := range msg.Runes {
					switch r {
					case 'j', 'J':
						if len(mode.filtered) > 0 {
							mode.cursor = intMin(mode.cursor+1, len(mode.filtered)-1)
						}
						m.mode = mode
						return m, nil
					case 'k', 'K':
						mode.cursor = intMax(mode.cursor-1, 0)
						m.mode = mode
						return m, nil
					}
				}
				// Delegate to filter input for typing
				newInput, cmd := mode.filter.Update(msg)
				mode.filter = newInput
				mode.filtered = filterCoins(mode.allCoins, mode.filter.Value())
				// Clamp cursor after filter
				switch {
				case len(mode.filtered) == 0:
					mode.cursor = 0
				case mode.cursor >= len(mode.filtered):
					mode.cursor = len(mode.filtered) - 1
				case mode.cursor < 0:
					mode.cursor = 0
				}
				m.mode = mode
				return m, cmd
			case tea.KeyDown:
				if len(mode.filtered) > 0 {
					mode.cursor = intMin(mode.cursor+1, len(mode.filtered)-1)
				}
				m.mode = mode
				return m, nil
			case tea.KeyUp:
				mode.cursor = intMax(mode.cursor-1, 0)
				m.mode = mode
				return m, nil
			default:
				// Delegate to filter input
				newInput, cmd := mode.filter.Update(msg)
				mode.filter = newInput
				mode.filtered = filterCoins(mode.allCoins, mode.filter.Value())
				switch {
				case len(mode.filtered) == 0:
					mode.cursor = 0
				case mode.cursor >= len(mode.filtered):
					mode.cursor = len(mode.filtered) - 1
				case mode.cursor < 0:
					mode.cursor = 0
				}
				m.mode = mode
				return m, cmd
			}

		case addAmount:
			switch msg.Type {
			case tea.KeyEsc:
				m.mode = mode.coinMode
				return m, nil
			case tea.KeyEnter:
				input := strings.TrimSpace(mode.input.Value())
				amount, err := strconv.ParseFloat(input, 64)
				if err != nil || amount <= 0 {
					mode.errMsg = "enter a positive number"
					m.mode = mode
					return m, nil
				}
				if len(m.portfolios) > 0 {
					return m, m.cmdUpsertHolding(m.portfolios[m.cursor].ID, mode.coin.ID, amount)
				}
				return m, nil
			default:
				// Delegate to amount input
				newInput, cmd := mode.input.Update(msg)
				mode.input = newInput
				m.mode = mode
				return m, cmd
			}
		}

	case portfoliosLoadedMsg:
		m.portfolios = msg.portfolios
		m.mode = browsing{}
		// Position cursor on focusID if provided
		if msg.focusID != 0 {
			for i, p := range m.portfolios {
				if p.ID == msg.focusID {
					m.cursor = i
					break
				}
			}
		}
		// Clamp cursor
		if m.cursor >= len(m.portfolios) {
			m.cursor = len(m.portfolios) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		// Load holdings for current portfolio
		if len(m.portfolios) > 0 {
			return m, m.cmdLoadHoldings(m.portfolios[m.cursor].ID)
		}
		return m, nil

	case coinPickerReadyMsg:
		// Build map of already-held coin IDs
		heldCoinIDs := make(map[int64]bool)
		for _, h := range m.holdings {
			heldCoinIDs[h.CoinID] = true
		}
		// Filter out held coins
		available := make([]store.Coin, 0)
		for _, c := range msg.coins {
			if !heldCoinIDs[c.ID] {
				available = append(available, c)
			}
		}
		if len(available) == 0 {
			m.lastErr = "all coins already in portfolio"
			return m, nil
		}
		// Enter addCoin mode
		ti := textinput.New()
		ti.Placeholder = "filter coins..."
		ti.CharLimit = 30
		ti.Focus()
		m.mode = addCoin{
			filter:   ti,
			allCoins: available,
			filtered: available,
			cursor:   0,
		}
		return m, nil

	case holdingsLoadedMsg:
		m.holdings = msg.holdings
		return m, nil

	case holdingsSavedMsg:
		m.holdings = msg.holdings
		m.mode = browsing{}
		return m, nil

	case errMsg:
		m.lastErr = msg.err.Error()
		return m, nil
	}

	return m, nil
}

// InputActive returns true when a text input is focused.
func (m PortfolioModel) InputActive() bool {
	switch m.mode.(type) {
	case creating, addCoin, addAmount:
		return true
	}
	return false
}

// View renders the two-panel layout with optional dialog overlay.
func (m PortfolioModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Dialog modes: creating, addCoin, addAmount
	switch mode := m.mode.(type) {
	case creating:
		dialog := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Render("New Portfolio\n\n" + mode.input.View())
		content := lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, dialog)
		return content + "\n" + m.renderStatusBar()

	case addCoin:
		dialog := m.renderAddCoinDialog(mode)
		content := lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, dialog)
		return content + "\n" + m.renderStatusBar()

	case addAmount:
		dialog := m.renderAddAmountDialog(mode)
		content := lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, dialog)
		return content + "\n" + m.renderStatusBar()
	}

	contentHeight := m.height - 3 // Reserve 1 row for status bar, 2 for borders

	// Define panel widths in terms of outer (total) columns and derive inner
	leftPanelOuter := 30
	rightPanelOuter := m.width - leftPanelOuter
	leftPanelInner := leftPanelOuter - 2
	rightPanelInner := rightPanelOuter - 2

	// Build left panel
	leftContent := m.renderLeftPanel(contentHeight, leftPanelInner)

	// Build right panel
	rightContent := m.renderRightPanel(contentHeight, rightPanelInner)

	// Combine panels with borders (no space separator — borders provide visual separation)
	leftStyle := lipgloss.NewStyle().
		Width(leftPanelInner).
		Height(contentHeight).
		Border(lipgloss.NormalBorder())
	rightStyle := lipgloss.NewStyle().
		Width(rightPanelInner).
		Height(contentHeight).
		Border(lipgloss.NormalBorder())

	panels := lipgloss.JoinHorizontal(lipgloss.Top,
		leftStyle.Render(leftContent),
		rightStyle.Render(rightContent),
	)

	// Status bar
	statusBar := m.renderStatusBar()

	return panels + "\n" + statusBar
}

func (m PortfolioModel) renderAddCoinDialog(mode addCoin) string {
	var b strings.Builder
	b.WriteString("Select Coin\n\n")
	b.WriteString(mode.filter.View() + "\n\n")

	// Show filtered coins
	for i, c := range mode.filtered {
		if i >= 10 { // Limit to 10 visible items
			_, _ = fmt.Fprintf(&b, "... and %d more\n", len(mode.filtered)-10)
			break
		}
		prefix := "  "
		if i == mode.cursor {
			prefix = "> "
		}
		_, _ = fmt.Fprintf(&b, "%s%s (%s)\n", prefix, c.Name, c.Ticker)
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Render(b.String())
}

func (m PortfolioModel) renderAddAmountDialog(mode addAmount) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "Add %s (%s)\n\n", mode.coin.Name, mode.coin.Ticker)
	b.WriteString("Amount: " + mode.input.View() + "\n")
	if mode.errMsg != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(mode.errMsg))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Render(b.String())
}

func (m PortfolioModel) renderLeftPanel(height, width int) string {
	titleStyle := lipgloss.NewStyle().Bold(true)
	title := titleStyle.Render("Portfolios")

	if len(m.portfolios) == 0 {
		return title + "\n" + "no portfolios — press n to create one"
	}

	highlight := lipgloss.NewStyle().Reverse(true)

	// Reserve 1 line for title
	contentHeight := height - 1
	var b strings.Builder
	for i, p := range m.portfolios {
		if i >= contentHeight {
			break
		}
		line := p.Name
		// Pad to full width for consistent highlight
		if len(line) < width {
			line += strings.Repeat(" ", width-len(line))
		}
		if i == m.cursor {
			line = highlight.Render(line)
		}
		b.WriteString(line + "\n")
	}
	return title + "\n" + b.String()
}

func (m PortfolioModel) renderRightPanel(height, width int) string {
	titleStyle := lipgloss.NewStyle().Bold(true)

	if len(m.portfolios) == 0 {
		return titleStyle.Render("Holdings") + "\n"
	}

	selectedPortfolio := m.portfolios[m.cursor]

	// Calculate total value
	var totalValue float64
	for _, h := range m.holdings {
		totalValue += h.Value
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Holdings — %s (%s)", selectedPortfolio.Name, format.FmtMoney(totalValue))) + "\n")

	if len(m.holdings) == 0 {
		b.WriteString("no holdings — press a to add one")
		return b.String()
	}

	// Header
	_, _ = fmt.Fprintf(&b, "%-15s %8s %10s %12s %12s %8s %6s\n", "Coin", "Ticker", "Amount", "Price", "Value", "24h", "%")
	b.WriteString(strings.Repeat("-", intMin(width, 80)) + "\n")

	// Rows
	for _, h := range m.holdings {
		changeStr := format.FmtChange(h.PriceChange)
		_, _ = fmt.Fprintf(&b, "%-15s %8s %10.4f %12s %12s %8s %5.1f%%\n",
			truncate(h.Name, 15),
			h.Ticker,
			h.Amount,
			format.FmtPrice(h.Rate),
			format.FmtMoney(h.Value),
			changeStr,
			h.Proportion,
		)
	}

	return b.String()
}

func (m PortfolioModel) renderStatusBar() string {
	var content string
	switch m.mode.(type) {
	case browsing:
		content = "j/k portfolios • a add holding • n new portfolio • q quit"
	case creating:
		content = "Enter to create • Esc to cancel"
	case addCoin:
		content = "j/k navigate • type to filter • Enter select • Esc cancel"
	case addAmount:
		content = "Enter to confirm • Esc back to coin selection"
	}

	if m.lastErr != "" {
		content += " • error: " + m.lastErr
	}

	return content
}

func (m *PortfolioModel) moveCursor(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.portfolios) {
		m.cursor = len(m.portfolios) - 1
	}
	if len(m.portfolios) == 0 {
		m.cursor = 0
	}
}

func (m PortfolioModel) openCreateDialog() PortfolioModel {
	ti := textinput.New()
	ti.Placeholder = "e.g. Long Term"
	ti.CharLimit = 50
	ti.Focus()
	m.mode = creating{input: ti}
	return m
}

func (m PortfolioModel) cmdLoadPortfolios() tea.Cmd {
	return func() tea.Msg {
		portfolios, err := m.store.GetAllPortfolios(m.ctx)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading portfolios: %w", err)}
		}
		return portfoliosLoadedMsg{portfolios: portfolios}
	}
}

func (m PortfolioModel) cmdCreatePortfolio(name string) tea.Cmd {
	return func() tea.Msg {
		p, err := m.store.CreatePortfolio(m.ctx, name)
		if err != nil {
			return errMsg{err: fmt.Errorf("creating portfolio: %w", err)}
		}
		portfolios, err := m.store.GetAllPortfolios(m.ctx)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading portfolios after create: %w", err)}
		}
		return portfoliosLoadedMsg{portfolios: portfolios, focusID: p.ID}
	}
}

// cmdOpenCoinPicker loads all coins from the store.
func (m PortfolioModel) cmdOpenCoinPicker() tea.Cmd {
	return func() tea.Msg {
		coins, err := m.store.GetAllCoins(m.ctx)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading coins for picker: %w", err)}
		}
		return coinPickerReadyMsg{coins: coins}
	}
}

// cmdLoadHoldings fetches holdings for the given portfolio ID.
func (m PortfolioModel) cmdLoadHoldings(portfolioID int64) tea.Cmd {
	return func() tea.Msg {
		holdings, err := m.store.GetHoldingsForPortfolio(m.ctx, portfolioID)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading holdings: %w", err)}
		}
		return holdingsLoadedMsg{holdings: holdings}
	}
}

// cmdUpsertHolding saves the holding and reloads the holdings list.
func (m PortfolioModel) cmdUpsertHolding(portfolioID, coinID int64, amount float64) tea.Cmd {
	return func() tea.Msg {
		if err := m.store.UpsertHolding(m.ctx, portfolioID, coinID, amount); err != nil {
			return errMsg{err: fmt.Errorf("saving holding: %w", err)}
		}
		holdings, err := m.store.GetHoldingsForPortfolio(m.ctx, portfolioID)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading holdings after save: %w", err)}
		}
		return holdingsSavedMsg{holdings: holdings}
	}
}

// transitionToAddAmount transitions from addCoin to addAmount mode.
func (m PortfolioModel) transitionToAddAmount(mode addCoin) (PortfolioModel, tea.Cmd) {
	if len(mode.filtered) == 0 {
		return m, nil
	}
	selectedCoin := mode.filtered[mode.cursor]
	ti := textinput.New()
	ti.Placeholder = "e.g. 0.5"
	ti.CharLimit = 20
	ti.Focus()
	m.mode = addAmount{
		coin:     selectedCoin,
		input:    ti,
		coinMode: mode,
	}
	return m, nil
}

// filterCoins filters coins by query (case-insensitive) matching name, ticker, or api_id.
func filterCoins(coins []store.Coin, query string) []store.Coin {
	if query == "" {
		return coins
	}
	q := strings.ToLower(query)
	result := make([]store.Coin, 0)
	for _, c := range coins {
		if strings.Contains(strings.ToLower(c.Name), q) ||
			strings.Contains(strings.ToLower(c.Ticker), q) ||
			strings.Contains(strings.ToLower(c.ApiID), q) {
			result = append(result, c)
		}
	}
	return result
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
