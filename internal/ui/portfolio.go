package ui

import (
	"context"
	"errors"
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
		origin   portfolioMode // track parent mode (browsing or listing) for return
	}
	addAmount struct {
		coin     store.Coin
		input    textinput.Model
		errMsg   string
		coinMode addCoin       // preserved so Esc returns to coin picker with state intact
		origin   portfolioMode // track parent mode (browsing or listing) for return
	}
	listing       struct{} // list mode: right panel focus, j/k navigate holdings
	editingAmount struct {
		holding  store.HoldingRow
		input    textinput.Model
		errMsg   string
		listMode listing // preserved so Esc returns to list mode with state intact
	}
	deleting struct {
		holding  store.HoldingRow
		listMode listing // preserved so Esc/cancel returns to list mode with state intact
	}
	editingPortfolio struct {
		portfolio store.Portfolio
		input     textinput.Model
		errMsg    string
	}
	deletingPortfolio struct {
		portfolio store.Portfolio
	}
)

func (browsing) isPortfolioMode()          {}
func (creating) isPortfolioMode()          {}
func (addCoin) isPortfolioMode()           {}
func (addAmount) isPortfolioMode()         {}
func (listing) isPortfolioMode()           {}
func (editingAmount) isPortfolioMode()     {}
func (deleting) isPortfolioMode()          {}
func (editingPortfolio) isPortfolioMode()  {}
func (deletingPortfolio) isPortfolioMode() {}

// portfoliosLoadedMsg is sent when portfolios are loaded from the store.
// focusID is non-zero when the cursor should be positioned on a specific portfolio.
type portfoliosLoadedMsg struct {
	portfolios []store.Portfolio
	focusID    int64
}

// coinPickerReadyMsg is sent when all coins are loaded for the picker.
type coinPickerReadyMsg struct {
	coins  []store.Coin
	origin portfolioMode // track parent mode (browsing or listing) for return
}

// holdingsLoadedMsg is sent when holdings are loaded for the current portfolio.
type holdingsLoadedMsg struct {
	holdings []store.HoldingRow
}

// holdingsSavedMsg is sent after a holding is successfully saved.
type holdingsSavedMsg struct {
	holdings []store.HoldingRow
}

// holdingDeletedMsg is sent after a holding is successfully deleted.
type holdingDeletedMsg struct {
	holdings []store.HoldingRow
}

// portfolioDeletedMsg is sent after a portfolio is successfully deleted.
type portfolioDeletedMsg struct {
	portfolios []store.Portfolio
}

// PortfolioModel is the Portfolio tab with two-panel layout.
type PortfolioModel struct {
	ctx            context.Context
	store          store.Store
	width          int
	height         int
	portfolios     []store.Portfolio
	cursor         int
	holdings       []store.HoldingRow
	holdingsCursor int // cursor position within holdings list (list mode)
	scrollOffset   int // vertical scroll offset for holdings list preview
	mode           portfolioMode
	lastErr        string
	currency       string
}

// NewPortfolioModel creates a new PortfolioModel with the given dependencies.
func NewPortfolioModel(ctx context.Context, s store.Store, currency string) PortfolioModel {
	return PortfolioModel{
		ctx:      ctx,
		store:    s,
		mode:     browsing{},
		currency: currency,
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
						m.scrollOffset = 0 // Reset scroll offset when navigating portfolios
						if len(m.portfolios) > 0 {
							return m, m.cmdLoadHoldings(m.portfolios[m.cursor].ID)
						}
						return m, nil
					case 'k', 'K':
						m.moveCursor(-1)
						m.scrollOffset = 0 // Reset scroll offset when navigating portfolios
						if len(m.portfolios) > 0 {
							return m, m.cmdLoadHoldings(m.portfolios[m.cursor].ID)
						}
						return m, nil
					case 'n', 'N':
						return m.openCreateDialog(), nil
					case 'e', 'E':
						if len(m.portfolios) > 0 {
							return m.openEditPortfolioDialog(), nil
						}
						return m, nil
					case 'X', 'x':
						if len(m.portfolios) > 0 {
							m.mode = deletingPortfolio{portfolio: m.portfolios[m.cursor]}
							return m, nil
						}
						return m, nil
					}
				}
			case tea.KeyDown:
				m.moveCursor(1)
				m.scrollOffset = 0 // Reset scroll offset when navigating portfolios
				if len(m.portfolios) > 0 {
					return m, m.cmdLoadHoldings(m.portfolios[m.cursor].ID)
				}
				return m, nil
			case tea.KeyUp:
				m.moveCursor(-1)
				m.scrollOffset = 0 // Reset scroll offset when navigating portfolios
				if len(m.portfolios) > 0 {
					return m, m.cmdLoadHoldings(m.portfolios[m.cursor].ID)
				}
				return m, nil
			case tea.KeyEnter:
				// Enter list mode if we have a portfolio (even if no holdings)
				if len(m.portfolios) > 0 {
					m.mode = listing{}
					m.holdingsCursor = 0
					m.scrollOffset = 0
				}
				return m, nil
			case tea.KeyPgDown:
				m.scrollOffset += m.visibleHoldingsRows() / 2
				m.clampScrollOffset()
				return m, nil
			case tea.KeyPgUp:
				m.scrollOffset -= m.visibleHoldingsRows() / 2
				m.clampScrollOffset()
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
				m.mode = mode.origin
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

		case listing:
			switch msg.Type {
			case tea.KeyRunes:
				for _, r := range msg.Runes {
					switch r {
					case 'j', 'J':
						m.holdingsCursor++
						m.clampHoldingsCursor()
						m.adjustHoldingsViewport()
						return m, nil
					case 'k', 'K':
						m.holdingsCursor--
						m.clampHoldingsCursor()
						m.adjustHoldingsViewport()
						return m, nil
					case 'g':
						m.holdingsCursor = 0
						m.adjustHoldingsViewport()
						return m, nil
					case 'G':
						if len(m.holdings) > 0 {
							m.holdingsCursor = len(m.holdings) - 1
						}
						m.adjustHoldingsViewport()
						return m, nil
					case 'a', 'A':
						// Open coin picker from list mode
						return m, m.cmdOpenCoinPickerFromList(mode)
					case 'X', 'x':
						// Open delete confirmation for current holding
						if len(m.holdings) > 0 && m.holdingsCursor < len(m.holdings) {
							m.mode = deleting{
								holding:  m.holdings[m.holdingsCursor],
								listMode: mode,
							}
						}
						return m, nil
					}
				}
			case tea.KeyDown:
				m.holdingsCursor++
				m.clampHoldingsCursor()
				m.adjustHoldingsViewport()
				return m, nil
			case tea.KeyUp:
				m.holdingsCursor--
				m.clampHoldingsCursor()
				m.adjustHoldingsViewport()
				return m, nil
			case tea.KeyEnter:
				// Open edit dialog for current holding
				if len(m.holdings) > 0 && m.holdingsCursor < len(m.holdings) {
					return m.openEditAmountDialog(m.holdings[m.holdingsCursor], mode)
				}
				return m, nil
			case tea.KeyEsc:
				m.mode = browsing{}
				m.holdingsCursor = 0
				m.scrollOffset = 0
				return m, nil
			}

		case editingAmount:
			switch msg.Type {
			case tea.KeyEsc:
				m.mode = mode.listMode
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
					return m, m.cmdUpdateHoldingAmount(m.portfolios[m.cursor].ID, mode.holding.CoinID, amount, mode.listMode)
				}
				return m, nil
			default:
				// Delegate to amount input
				newInput, cmd := mode.input.Update(msg)
				mode.input = newInput
				m.mode = mode
				return m, cmd
			}

		case deleting:
			switch msg.Type {
			case tea.KeyEsc:
				m.mode = mode.listMode
				return m, nil
			case tea.KeyEnter:
				// Delete the holding
				if len(m.portfolios) > 0 {
					return m, m.cmdDeleteHolding(m.portfolios[m.cursor].ID, mode.holding.ID, mode.listMode)
				}
				return m, nil
			}
			// All other keys ignored in deleting mode

		case editingPortfolio:
			switch msg.Type {
			case tea.KeyEsc:
				m.mode = browsing{}
				return m, nil
			case tea.KeyEnter:
				name := strings.TrimSpace(mode.input.Value())
				if name == "" || name == mode.portfolio.Name {
					m.mode = browsing{}
					return m, nil
				}
				for _, p := range m.portfolios {
					if p.Name == name && p.ID != mode.portfolio.ID {
						mode.errMsg = "name already exists"
						m.mode = mode
						return m, nil
					}
				}
				return m, m.cmdRenamePortfolio(mode.portfolio.ID, name)
			default:
				newInput, cmd := mode.input.Update(msg)
				mode.input = newInput
				m.mode = mode
				return m, cmd
			}

		case deletingPortfolio:
			switch msg.Type {
			case tea.KeyEsc:
				m.mode = browsing{}
				return m, nil
			case tea.KeyEnter:
				return m, m.cmdDeletePortfolio(mode.portfolio.ID)
			}
			// All other keys ignored
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
		// Check if no coins are loaded in the database
		if len(msg.coins) == 0 {
			m.lastErr = "no coins loaded — visit Markets tab first"
			return m, nil
		}

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
		// Enter addCoin mode with origin preserved
		m.lastErr = ""
		ti := textinput.New()
		ti.Placeholder = "filter coins..."
		ti.CharLimit = 30
		ti.Focus()
		m.mode = addCoin{
			filter:   ti,
			allCoins: available,
			filtered: available,
			cursor:   0,
			origin:   msg.origin,
		}
		return m, nil

	case holdingsLoadedMsg:
		m.holdings = msg.holdings
		return m, nil

	case holdingsSavedMsg:
		m.holdings = msg.holdings
		// Determine return mode based on the mode we were in
		switch prevMode := m.mode.(type) {
		case editingAmount:
			// Editing always returns to listing
			m.mode = listing{}
			m.clampHoldingsCursor()
		case addAmount:
			// Adding returns to listing if we came from listing mode, else browsing
			if _, fromListing := prevMode.origin.(listing); fromListing {
				m.mode = listing{}
				m.clampHoldingsCursor()
			} else {
				m.mode = browsing{}
			}
		default:
			m.mode = browsing{}
		}
		return m, nil

	case holdingDeletedMsg:
		m.holdings = msg.holdings
		// Clamp cursor if needed
		m.clampHoldingsCursor()
		// Return to listing mode, or browsing if holdings are now empty
		if len(m.holdings) == 0 {
			m.mode = browsing{}
			m.holdingsCursor = 0
			m.scrollOffset = 0
		} else {
			m.mode = listing{}
		}
		return m, nil

	case portfolioDeletedMsg:
		m.portfolios = msg.portfolios
		m.mode = browsing{}
		m.scrollOffset = 0
		m.holdingsCursor = 0
		if len(m.portfolios) == 0 {
			m.cursor = 0
			m.holdings = nil
			m.lastErr = ""
			return m, nil
		}
		if m.cursor >= len(m.portfolios) {
			m.cursor = len(m.portfolios) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.lastErr = ""
		return m, m.cmdLoadHoldings(m.portfolios[m.cursor].ID)

	case errMsg:
		m.lastErr = msg.err.Error()
		return m, nil

	case currencyChangedMsg:
		m.currency = msg.code
		// Don't reload holdings yet — prices in DB are still in the old currency.
		// When MarketsModel finishes refreshing, pricesUpdatedMsg will be broadcast,
		// and we'll reload holdings then.
		return m, nil

	case pricesUpdatedMsg:
		// Prices in the DB have been updated (possibly in a new currency after a
		// currency change). Reload holdings so Value, Proportion, and totals
		// reflect the new rates.
		if len(m.portfolios) > 0 {
			return m, m.cmdLoadHoldings(m.portfolios[m.cursor].ID)
		}
		return m, nil
	}

	return m, nil
}

// InputActive returns true when a text input is focused.
func (m PortfolioModel) InputActive() bool {
	switch m.mode.(type) {
	case creating, addCoin, addAmount, editingAmount, editingPortfolio:
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
		dialogWidth := intMin(intMax(m.width/2, 40), 60)
		inputWidth := dialogWidth - 6 // Account for padding and border
		mode.input.Width = inputWidth
		dialog := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Width(dialogWidth).
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

	case editingAmount:
		dialog := m.renderEditAmountDialog(mode)
		content := lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, dialog)
		return content + "\n" + m.renderStatusBar()

	case deleting:
		dialog := m.renderDeleteDialog(mode)
		content := lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, dialog)
		return content + "\n" + m.renderStatusBar()

	case editingPortfolio:
		dialog := m.renderEditPortfolioDialog(mode)
		content := lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, dialog)
		return content + "\n" + m.renderStatusBar()

	case deletingPortfolio:
		dialog := m.renderDeletePortfolioDialog(mode)
		content := lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, dialog)
		return content + "\n" + m.renderStatusBar()
	}

	contentHeight := m.height - 3 // Reserve 1 row for status bar, 2 for borders

	// Define panel widths in terms of outer (total) columns and derive inner.
	// Left panel is kept narrow (22) so the right panel has enough room for all
	// holdings columns (77 chars) without wrapping.
	leftPanelOuter := 22
	rightPanelOuter := m.width - leftPanelOuter
	leftPanelInner := leftPanelOuter - 2
	rightPanelInner := rightPanelOuter - 2

	// Build left panel
	leftContent := m.renderLeftPanel(contentHeight, leftPanelInner)

	// Build right panel
	rightContent := m.renderRightPanel(contentHeight, rightPanelInner)

	// Determine which panel is focused based on mode
	leftFocused := false
	rightFocused := false
	switch m.mode.(type) {
	case browsing, creating, addCoin, addAmount, editingPortfolio, deletingPortfolio:
		leftFocused = true
	case listing, editingAmount, deleting:
		// When in dialog modes from list mode, right panel stays focused
		rightFocused = true
	}

	// Combine panels with borders (no space separator — borders provide visual separation)
	// Focused panel gets accent border color, unfocused gets dim border color
	accentColor := lipgloss.Color("#00FFFF") // cyan accent
	dimColor := lipgloss.Color("#555555")    // dim gray

	leftBorderColor := dimColor
	if leftFocused {
		leftBorderColor = accentColor
	}
	rightBorderColor := dimColor
	if rightFocused {
		rightBorderColor = accentColor
	}

	leftStyle := lipgloss.NewStyle().
		Width(leftPanelInner).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(leftBorderColor)
	rightStyle := lipgloss.NewStyle().
		Width(rightPanelInner).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(rightBorderColor)

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
	b.WriteString(titleStyle.Render(fmt.Sprintf("Holdings — %s (%s)", selectedPortfolio.Name, format.FmtMoney(totalValue, m.currency))) + "\n")

	if len(m.holdings) == 0 {
		b.WriteString("no holdings — press a to add one")
		return b.String()
	}

	// Header
	currencyUpper := strings.ToUpper(m.currency)
	_, _ = fmt.Fprintf(&b, "%-15s %8s %10s %12s %12s %8s %6s\n",
		"Coin", "Ticker", "Amount", "Price ("+currencyUpper+")", "Value ("+currencyUpper+")", "24h", "%")
	b.WriteString(strings.Repeat("-", intMin(width, 80)) + "\n")

	// Determine which rows to show based on mode and scroll offset
	startIdx := 0
	endIdx := len(m.holdings)
	visibleRows := height - 3 // Account for title, header, separator

	_, isListing := m.mode.(listing)
	_, isEditing := m.mode.(editingAmount)
	_, isDeleting := m.mode.(deleting)
	inListMode := isListing || isEditing || isDeleting

	if inListMode && visibleRows > 0 {
		startIdx = m.scrollOffset
		endIdx = intMin(startIdx+visibleRows, len(m.holdings))
	} else if visibleRows > 0 {
		startIdx = m.scrollOffset
		endIdx = intMin(startIdx+visibleRows, len(m.holdings))
	}

	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > len(m.holdings) {
		endIdx = len(m.holdings)
	}

	// Rows
	highlight := lipgloss.NewStyle().Reverse(true)
	for i := startIdx; i < endIdx; i++ {
		h := m.holdings[i]
		changeStr := format.FmtChange(h.PriceChange)
		line := fmt.Sprintf("%-15s %8s %10.4f %12s %12s %8s %5.1f%%",
			truncate(h.Name, 15), h.Ticker, h.Amount,
			format.FmtPriceValue(h.Rate), format.FmtMoneyValue(h.Value),
			changeStr, h.Proportion)
		// Clip or pad to width before applying ANSI reverse styling so the outer
		// panel never word-wraps the highlighted row at an unexpected position.
		if inListMode && i == m.holdingsCursor {
			lw := lipgloss.Width(line)
			switch {
			case lw > width:
				runes := []rune(line)
				line = string(runes[:width])
			case lw < width:
				line += strings.Repeat(" ", width-lw)
			}
			line = highlight.Render(line)
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}

func (m PortfolioModel) renderStatusBar() string {
	var hints string
	switch m.mode.(type) {
	case browsing:
		hints = "j/k portfolios • Enter list • e edit • X delete • PgUp/PgDn scroll • n new • q quit"
	case creating:
		hints = "Enter to create • Esc to cancel"
	case addCoin:
		hints = "j/k navigate • type to filter • Enter select • Esc cancel"
	case addAmount:
		hints = "Enter to confirm • Esc back to coin selection"
	case listing:
		hints = "j/k holdings • g/G top/bottom • Enter edit • X delete • a add holding • Esc back to menu • q quit"
	case editingAmount:
		hints = "Enter to save • Esc cancel"
	case deleting:
		hints = "Enter to confirm delete • Esc cancel"
	case editingPortfolio:
		hints = "Enter to save • Esc to cancel"
	case deletingPortfolio:
		hints = "Enter to delete • Esc to cancel"
	}

	var right string
	if m.lastErr != "" {
		right = statusBarRed.Render("error: " + m.lastErr)
	}

	return renderStatusBar(m.width, hints, right)
}

func (m PortfolioModel) openEditPortfolioDialog() PortfolioModel {
	ti := textinput.New()
	ti.Placeholder = "e.g. Long Term"
	ti.CharLimit = 50
	ti.Focus()
	ti.SetValue(m.portfolios[m.cursor].Name)
	// Set width based on dialog width calculation (will be set properly in View)
	dialogWidth := intMin(intMax(m.width/2, 40), 60)
	ti.Width = dialogWidth - 6 // Account for padding and border
	m.mode = editingPortfolio{
		portfolio: m.portfolios[m.cursor],
		input:     ti,
	}
	return m
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
			if strings.Contains(err.Error(), "UNIQUE") {
				return errMsg{err: errors.New("portfolio \"" + name + "\" already exists")}
			}
			return errMsg{err: fmt.Errorf("creating portfolio: %w", err)}
		}
		portfolios, err := m.store.GetAllPortfolios(m.ctx)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading portfolios after create: %w", err)}
		}
		return portfoliosLoadedMsg{portfolios: portfolios, focusID: p.ID}
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
		origin:   mode.origin,
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

// visibleHoldingsRows returns how many holdings rows can be displayed.
func (m PortfolioModel) visibleHoldingsRows() int {
	// Account for: header (1) + separator (1) + title (1) + border (2)
	contentHeight := m.height - 5
	if contentHeight < 0 {
		return 0
	}
	return contentHeight
}

// clampScrollOffset ensures scrollOffset stays within valid bounds.
func (m *PortfolioModel) clampScrollOffset() {
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	maxOffset := len(m.holdings) - m.visibleHoldingsRows()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
}

// clampHoldingsCursor ensures holdingsCursor stays within valid bounds.
func (m *PortfolioModel) clampHoldingsCursor() {
	if m.holdingsCursor < 0 {
		m.holdingsCursor = 0
	}
	if len(m.holdings) == 0 {
		m.holdingsCursor = 0
	} else if m.holdingsCursor >= len(m.holdings) {
		m.holdingsCursor = len(m.holdings) - 1
	}
}

// adjustHoldingsViewport ensures holdingsCursor stays visible.
func (m *PortfolioModel) adjustHoldingsViewport() {
	visibleRows := m.visibleHoldingsRows()
	if visibleRows <= 0 {
		return
	}
	// If cursor is above the visible area, scroll up
	if m.holdingsCursor < m.scrollOffset {
		m.scrollOffset = m.holdingsCursor
	}
	// If cursor is below the visible area, scroll down
	if m.holdingsCursor >= m.scrollOffset+visibleRows {
		m.scrollOffset = m.holdingsCursor - visibleRows + 1
	}
	m.clampScrollOffset()
}

// cmdOpenCoinPickerFromList loads all coins from the store, preserving list mode for return.
func (m PortfolioModel) cmdOpenCoinPickerFromList(listMode listing) tea.Cmd {
	return func() tea.Msg {
		coins, err := m.store.GetAllCoins(m.ctx)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading coins for picker: %w", err)}
		}
		return coinPickerReadyMsg{coins: coins, origin: listMode}
	}
}

// openEditAmountDialog opens the edit amount dialog for a holding.
func (m PortfolioModel) openEditAmountDialog(holding store.HoldingRow, listMode listing) (PortfolioModel, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "e.g. 0.5"
	ti.CharLimit = 20
	ti.Focus()
	// Pre-populate with current amount (4 decimal places)
	ti.SetValue(fmt.Sprintf("%.4f", holding.Amount))
	m.mode = editingAmount{
		holding:  holding,
		input:    ti,
		listMode: listMode,
	}
	return m, nil
}

// cmdUpdateHoldingAmount updates a holding's amount and reloads holdings, preserving list mode.
func (m PortfolioModel) cmdUpdateHoldingAmount(portfolioID, coinID int64, amount float64, listMode listing) tea.Cmd {
	return func() tea.Msg {
		if err := m.store.UpsertHolding(m.ctx, portfolioID, coinID, amount); err != nil {
			return errMsg{err: fmt.Errorf("updating holding amount: %w", err)}
		}
		holdings, err := m.store.GetHoldingsForPortfolio(m.ctx, portfolioID)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading holdings after update: %w", err)}
		}
		return holdingsSavedMsg{holdings: holdings}
	}
}

// cmdDeleteHolding deletes a holding and reloads holdings, preserving list mode.
func (m PortfolioModel) cmdDeleteHolding(portfolioID, holdingID int64, listMode listing) tea.Cmd {
	return func() tea.Msg {
		if err := m.store.DeleteHolding(m.ctx, holdingID); err != nil {
			return errMsg{err: fmt.Errorf("deleting holding: %w", err)}
		}
		holdings, err := m.store.GetHoldingsForPortfolio(m.ctx, portfolioID)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading holdings after delete: %w", err)}
		}
		return holdingDeletedMsg{holdings: holdings}
	}
}

// cmdRenamePortfolio renames a portfolio and reloads portfolios.
func (m PortfolioModel) cmdRenamePortfolio(portfolioID int64, name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.store.RenamePortfolio(m.ctx, portfolioID, name); err != nil {
			return errMsg{err: fmt.Errorf("renaming portfolio: %w", err)}
		}
		portfolios, err := m.store.GetAllPortfolios(m.ctx)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading portfolios after rename: %w", err)}
		}
		return portfoliosLoadedMsg{portfolios: portfolios, focusID: portfolioID}
	}
}

// cmdDeletePortfolio deletes a portfolio and reloads portfolios.
func (m PortfolioModel) cmdDeletePortfolio(portfolioID int64) tea.Cmd {
	return func() tea.Msg {
		if err := m.store.DeletePortfolio(m.ctx, portfolioID); err != nil {
			return errMsg{err: fmt.Errorf("deleting portfolio: %w", err)}
		}
		portfolios, err := m.store.GetAllPortfolios(m.ctx)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading portfolios after delete: %w", err)}
		}
		return portfolioDeletedMsg{portfolios: portfolios}
	}
}

// renderEditAmountDialog renders the edit amount dialog.
func (m PortfolioModel) renderEditAmountDialog(mode editingAmount) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "Edit %s (%s)\n\n", mode.holding.Name, mode.holding.Ticker)
	_, _ = fmt.Fprintf(&b, "Current: %.4f\n", mode.holding.Amount)
	b.WriteString("New amount: " + mode.input.View() + "\n")
	if mode.errMsg != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(mode.errMsg))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Render(b.String())
}

// renderDeleteDialog renders the delete confirmation dialog.
func (m PortfolioModel) renderDeleteDialog(mode deleting) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "Delete Holding\n\n")
	_, _ = fmt.Fprintf(&b, "Coin: %s (%s)\n", mode.holding.Name, mode.holding.Ticker)
	_, _ = fmt.Fprintf(&b, "Amount: %.4f\n", mode.holding.Amount)
	_, _ = fmt.Fprintf(&b, "Value: %s\n\n", format.FmtMoney(mode.holding.Value, m.currency))
	b.WriteString("Press Enter to confirm deletion, or Esc to cancel.")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Render(b.String())
}

// renderEditPortfolioDialog renders the edit portfolio dialog.
func (m PortfolioModel) renderEditPortfolioDialog(mode editingPortfolio) string {
	var b strings.Builder
	b.WriteString("Rename Portfolio\n\n")
	b.WriteString(mode.input.View() + "\n")
	if mode.errMsg != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(mode.errMsg))
	}
	dialogWidth := intMin(intMax(m.width/2, 40), 60)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(dialogWidth).
		Render(b.String())
}

// renderDeletePortfolioDialog renders the delete portfolio confirmation dialog.
func (m PortfolioModel) renderDeletePortfolioDialog(mode deletingPortfolio) string {
	var b strings.Builder
	b.WriteString("Delete Portfolio\n\n")
	b.WriteString(mode.portfolio.Name + "\n\n")
	b.WriteString("Press Enter to confirm, or Esc to cancel.")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Render(b.String())
}
