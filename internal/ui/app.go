package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredericomozzato/crypto_tracker/internal/api"
	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

type tab int

const (
	tabMarkets tab = iota
	tabPortfolio
)

const tabCount = 2

func tabBarActiveStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color("#4A4A4A")).
		Foreground(lipgloss.Color("#FFFFFF"))
}

func tabBarInactiveStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#2A2A2A")).
		Foreground(lipgloss.Color("#888888"))
}

// AppModel is the root Bubble Tea model. Owns tab bar, tab routing, global quit.
type AppModel struct {
	width     int
	height    int
	activeTab tab
	markets   MarketsModel
	portfolio PortfolioModel
}

// NewAppModel creates a new AppModel with the given dependencies.
func NewAppModel(ctx context.Context, s store.Store, c api.CoinGeckoClient) AppModel {
	return AppModel{
		activeTab: tabMarkets,
		markets:   NewMarketsModel(ctx, s, c),
		portfolio: NewPortfolioModel(ctx, s),
	}
}

// Init delegates to both children models' Init commands.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(m.markets.Init(), m.portfolio.Init())
}

// Update handles WindowSizeMsg (propagated to children with height-1),
// tab switching keys (Tab/Shift+Tab/1/2), global quit (q/Ctrl+C),
// and delegates all other messages to the active child model.
// Tab switching is suppressed when the active child's InputActive() returns true.
// Ctrl+C always quits regardless of InputActive().
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		childMsg := tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height - 1}
		var cmd1, cmd2 tea.Cmd
		m.markets, cmd1 = m.markets.update(childMsg)
		m.portfolio, cmd2 = m.portfolio.update(childMsg)
		return m, tea.Batch(cmd1, cmd2)

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if !m.activeInputActive() {
			switch msg.Type {
			case tea.KeyTab:
				m.activeTab = tab((int(m.activeTab) + 1) % tabCount)
				return m, nil
			case tea.KeyShiftTab:
				m.activeTab = tab((int(m.activeTab) - 1 + tabCount) % tabCount)
				return m, nil
			case tea.KeyRunes:
				for _, r := range msg.Runes {
					switch r {
					case 'q':
						return m, tea.Quit
					case '1':
						m.activeTab = tabMarkets
						return m, nil
					case '2':
						m.activeTab = tabPortfolio
						return m, nil
					}
				}
			}
		}
	}

	// Background messages (non-key, non-resize) are always forwarded to both
	// children via tea.Batch. This ensures that async responses like
	// coinsLoadedMsg and pricesUpdatedMsg reach whichever tab issued the
	// command, even if the user has since switched tabs. Without this
	// broadcast, responses would be silently dropped when the inactive tab
	// doesn't match the issuing tab.
	var cmd1, cmd2 tea.Cmd
	m.markets, cmd1 = m.markets.update(msg)
	m.portfolio, cmd2 = m.portfolio.update(msg)
	return m, tea.Batch(cmd1, cmd2)
}

// View renders the tab bar + active child view.
func (m AppModel) View() string {
	tabBar := m.renderTabBar()
	switch m.activeTab {
	case tabMarkets:
		return tabBar + "\n" + m.markets.View()
	case tabPortfolio:
		return tabBar + "\n" + m.portfolio.View()
	}
	return tabBar
}

// renderTabBar renders "[ Markets ]  [ Portfolio ]" with active tab highlighted.
func (m AppModel) renderTabBar() string {
	inactiveStyle := tabBarInactiveStyle()
	activeStyle := tabBarActiveStyle()

	marketsLabel := " Markets "
	portfolioLabel := " Portfolio "

	if m.activeTab == tabMarkets {
		marketsLabel = activeStyle.Render(marketsLabel)
		portfolioLabel = inactiveStyle.Render(portfolioLabel)
	} else {
		marketsLabel = inactiveStyle.Render(marketsLabel)
		portfolioLabel = activeStyle.Render(portfolioLabel)
	}

	return marketsLabel + "  " + portfolioLabel
}

// activeInputActive returns whether the currently active child has a text input focused.
func (m AppModel) activeInputActive() bool {
	switch m.activeTab {
	case tabMarkets:
		return m.markets.InputActive()
	case tabPortfolio:
		return m.portfolio.InputActive()
	}
	return false
}
