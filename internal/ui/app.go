package ui

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredericomozzato/crypto_tracker/internal/api"
	"github.com/fredericomozzato/crypto_tracker/internal/format"
	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

// AppModel is the root Bubble Tea model for the crypto-tracker TUI.
type AppModel struct {
	width      int
	height     int
	ctx        context.Context
	store      store.Store
	client     api.CoinGeckoClient
	coins      []store.Coin
	lastErr    string
	refreshing bool
	cursor     int // index of selected row in m.coins
	offset     int // index of first visible row (viewport scroll)
}

// coinsLoadedMsg is sent when coins are successfully loaded from the API.
type coinsLoadedMsg struct {
	coins []store.Coin
}

// errMsg is sent when an error occurs during data fetching.
type errMsg struct {
	err error
}

// pricesUpdatedMsg is sent when prices are successfully refreshed.
type pricesUpdatedMsg struct {
	coins []store.Coin
}

// NewAppModel creates a new AppModel with the given dependencies.
func NewAppModel(ctx context.Context, s store.Store, c api.CoinGeckoClient) AppModel {
	return AppModel{
		ctx:    ctx,
		store:  s,
		client: c,
	}
}

// Init is the Bubble Tea init command. Fetches initial coin data.
func (m AppModel) Init() tea.Cmd {
	return func() tea.Msg {
		existing, err := m.store.GetAllCoins(m.ctx)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading coins: %w", err)}
		}
		if len(existing) > 0 {
			return coinsLoadedMsg{coins: existing}
		}

		fetched, err := m.client.FetchMarkets(m.ctx, 100)
		if err != nil {
			return errMsg{err: err}
		}
		for _, c := range fetched {
			if err := m.store.UpsertCoin(m.ctx, c); err != nil {
				return errMsg{err: fmt.Errorf("upserting coin %s: %w", c.ApiID, err)}
			}
		}
		stored, err := m.store.GetAllCoins(m.ctx)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading coins after seed: %w", err)}
		}
		return coinsLoadedMsg{coins: stored}
	}
}

// Update handles Bubble Tea messages.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyRunes:
			for _, r := range msg.Runes {
				switch r {
				case 'q':
					return m, tea.Quit
				case 'r':
					if !m.refreshing && len(m.coins) > 0 {
						m.refreshing = true
						return m, m.cmdRefresh()
					}
				case 'j':
					m.moveCursor(+1)
				case 'k':
					m.moveCursor(-1)
				case 'g':
					m.cursor = 0
					m.adjustViewport()
				case 'G':
					if len(m.coins) > 0 {
						m.cursor = len(m.coins) - 1
						m.adjustViewport()
					}
				}
			}
		case tea.KeyDown:
			m.moveCursor(+1)
		case tea.KeyUp:
			m.moveCursor(-1)
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case coinsLoadedMsg:
		m.coins = msg.coins
		m.lastErr = ""
		if m.cursor >= len(m.coins) && len(m.coins) > 0 {
			m.cursor = len(m.coins) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
	case pricesUpdatedMsg:
		m.coins = msg.coins
		m.refreshing = false
		m.lastErr = ""
	case errMsg:
		m.lastErr = msg.err.Error()
		m.refreshing = false
	}

	return m, nil
}

// cmdRefresh returns a command that refreshes prices for all loaded coins.
func (m AppModel) cmdRefresh() tea.Cmd {
	return func() tea.Msg {
		// Build list of API IDs from loaded coins
		apiIDs := make([]string, len(m.coins))
		for i, c := range m.coins {
			apiIDs[i] = c.ApiID
		}

		// Fetch fresh prices
		prices, err := m.client.FetchPrices(m.ctx, apiIDs)
		if err != nil {
			return errMsg{err: err}
		}

		// Update prices in store
		if err := m.store.UpdatePrices(m.ctx, prices); err != nil {
			return errMsg{err: err}
		}

		// Read back updated coins
		updatedCoins, err := m.store.GetAllCoins(m.ctx)
		if err != nil {
			return errMsg{err: err}
		}

		return pricesUpdatedMsg{coins: updatedCoins}
	}
}

// View renders the current state of the app.
func (m AppModel) View() string {
	// Check minimum terminal size
	if m.width < 100 || m.height < 30 {
		return "Terminal too small — resize to at least 100×30"
	}

	switch {
	case m.lastErr != "":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Render("Error: " + m.lastErr)
	case len(m.coins) == 0:
		return "loading..."
	}

	h := m.tableHeight()
	end := m.offset + h
	if end > len(m.coins) {
		end = len(m.coins)
	}

	// Column widths
	wRank := 4
	wName := 22
	wTicker := 8
	wPrice := 14
	wChange := 9

	highlight := lipgloss.NewStyle().Reverse(true)
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))

	// Header
	header := fmt.Sprintf(
		"%*s  %-*s  %-*s  %*s  %*s",
		wRank, "#",
		wName, "Name",
		wTicker, "Ticker",
		wPrice, "Price (USD)",
		wChange, "24h",
	)

	var lines []string
	lines = append(lines, header)

	for i := m.offset; i < end; i++ {
		c := m.coins[i]
		price := format.FmtPrice(c.Rate)
		change := format.FmtChange(c.PriceChange)

		if c.PriceChange >= 0 {
			change = green.Render(change)
		} else {
			change = red.Render(change)
		}

		line := fmt.Sprintf(
			"%*d  %-*s  %-*s  %*s  %*s",
			wRank, c.MarketRank,
			wName, truncate(c.Name, wName-2),
			wTicker, c.Ticker,
			wPrice, price,
			wChange, change,
		)

		if i == m.cursor {
			line = highlight.Render(line)
		}

		lines = append(lines, line)
	}

	// Pad remaining rows
	for len(lines)-1 < h {
		lines = append(lines, "")
	}

	// Hint line
	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render("j/k navigate • g/G top/bottom • r refresh • q quit")

	return strings.Join(lines, "\n") + "\n" + hint
}

// moveCursor moves the cursor by delta and adjusts the viewport.
func (m *AppModel) moveCursor(delta int) {
	if len(m.coins) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.coins) {
		m.cursor = len(m.coins) - 1
	}
	m.adjustViewport()
}

// adjustViewport updates m.offset so the cursor row stays visible.
func (m *AppModel) adjustViewport() {
	h := m.tableHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
	maxOff := len(m.coins) - h
	if maxOff < 0 {
		maxOff = 0
	}
	if m.offset > maxOff {
		m.offset = maxOff
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// tableHeight returns the number of rows available for coin data.
// Reserves 1 row for column headers and 1 row for the hint line.
func (m AppModel) tableHeight() int {
	h := m.height - 2
	if h < 1 {
		return 1
	}
	return h
}

// truncate returns s truncated to maxLen characters with an ellipsis.
func truncate(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	runes := []rune(s)
	return string(runes[:maxLen-1]) + "…"
}
