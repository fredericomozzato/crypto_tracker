package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredericomozzato/crypto_tracker/internal/api"
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
		// Fetch one coin from the API
		coins, err := m.client.FetchMarkets(m.ctx, 1)
		if err != nil {
			return errMsg{err: err}
		}

		// Upsert the fetched coin(s) into the store
		for _, coin := range coins {
			if err := m.store.UpsertCoin(m.ctx, coin); err != nil {
				return errMsg{err: err}
			}
		}

		// Read back all coins from the store
		storedCoins, err := m.store.GetAllCoins(m.ctx)
		if err != nil {
			return errMsg{err: err}
		}

		return coinsLoadedMsg{coins: storedCoins}
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
				if r == 'q' {
					return m, tea.Quit
				}
				if r == 'r' && !m.refreshing && len(m.coins) > 0 {
					m.refreshing = true
					return m, m.cmdRefresh()
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case coinsLoadedMsg:
		m.coins = msg.coins
		m.lastErr = ""
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

	var content string

	switch {
	case m.lastErr != "":
		content = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Render("Error: " + m.lastErr)
	case len(m.coins) > 0:
		// Display the first coin
		c := m.coins[0]
		content = fmt.Sprintf(
			"%s (%s)\nPrice: $%.2f\n24h Change: %.2f%%\n\n%s",
			c.Name,
			c.Ticker,
			c.Rate,
			c.PriceChange,
			m.refreshHint(),
		)
	default:
		content = "loading..."
	}

	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)

	return style.Render(content)
}

// refreshHint returns the hint text for refreshing prices.
func (m AppModel) refreshHint() string {
	if m.refreshing {
		return "refreshing..."
	}
	return "r to refresh"
}
