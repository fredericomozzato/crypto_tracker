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
	width  int
	height int
	store  store.Store
	client api.CoinGeckoClient
	coins  []store.Coin
	errMsg string
}

// coinsLoadedMsg is sent when coins are successfully loaded from the API.
type coinsLoadedMsg struct {
	coins []store.Coin
}

// errMsg is sent when an error occurs during data fetching.
type errMsg struct {
	err error
}

// NewAppModel creates a new AppModel with the given dependencies.
func NewAppModel(s store.Store, c api.CoinGeckoClient) AppModel {
	return AppModel{
		store:  s,
		client: c,
	}
}

// Init is the Bubble Tea init command. Fetches initial coin data.
func (m AppModel) Init() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Fetch one coin from the API
		coins, err := m.client.FetchMarkets(ctx, 1)
		if err != nil {
			return errMsg{err: err}
		}

		// Upsert the fetched coin(s) into the store
		for _, coin := range coins {
			if err := m.store.UpsertCoin(ctx, coin); err != nil {
				return errMsg{err: err}
			}
		}

		// Read back all coins from the store
		storedCoins, err := m.store.GetAllCoins(ctx)
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
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case coinsLoadedMsg:
		m.coins = msg.coins
	case errMsg:
		m.errMsg = msg.err.Error()
	}

	return m, nil
}

// View renders the current state of the app.
func (m AppModel) View() string {
	// Check minimum terminal size
	if m.width < 80 || m.height < 24 {
		return "Terminal too small — resize to at least 80×24"
	}

	var content string

	switch {
	case m.errMsg != "":
		content = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Render("Error: " + m.errMsg)
	case len(m.coins) > 0:
		// Display the first coin
		c := m.coins[0]
		content = fmt.Sprintf(
			"%s (%s)\nPrice: $%.2f\n24h Change: %.2f%%",
			c.Name,
			c.Ticker,
			c.Rate,
			c.PriceChange,
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
