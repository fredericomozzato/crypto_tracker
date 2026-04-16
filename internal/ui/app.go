package ui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredericomozzato/crypto_tracker/internal/api"
	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

type tab int

const (
	tabMarkets tab = iota
	tabPortfolio
	tabSettings
)

const tabCount = 3

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
	settings  SettingsModel
}

// NewAppModel creates a new AppModel with the given dependencies.
func NewAppModel(ctx context.Context, s store.Store, c api.CoinGeckoClient) AppModel {
	return AppModel{
		activeTab: tabMarkets,
		markets:   NewMarketsModel(ctx, s, c),
		portfolio: NewPortfolioModel(ctx, s),
		settings:  NewSettingsModel(ctx, s),
	}
}

// Init delegates to all children models' Init commands.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(m.markets.Init(), m.portfolio.Init(), m.settings.Init())
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
		var cmd1, cmd2, cmd3 tea.Cmd
		m.markets, cmd1 = m.markets.update(childMsg)
		m.portfolio, cmd2 = m.portfolio.update(childMsg)
		m.settings, cmd3 = m.settings.update(childMsg)
		return m, tea.Batch(cmd1, cmd2, cmd3)

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
					case '3':
						m.activeTab = tabSettings
						return m, nil
					}
				}
			}
		}
	}

	// Background messages (non-key, non-resize) are always forwarded to all
	// children via tea.Batch. This ensures that async responses reach whichever
	// tab issued the command, even if the user has since switched tabs.
	var cmd1, cmd2, cmd3 tea.Cmd
	m.markets, cmd1 = m.markets.update(msg)
	m.portfolio, cmd2 = m.portfolio.update(msg)
	m.settings, cmd3 = m.settings.update(msg)
	return m, tea.Batch(cmd1, cmd2, cmd3)
}

// View renders the tab bar + active child view.
func (m AppModel) View() string {
	tabBar := m.renderTabBar()
	switch m.activeTab {
	case tabMarkets:
		return tabBar + "\n" + m.markets.View()
	case tabPortfolio:
		return tabBar + "\n" + m.portfolio.View()
	case tabSettings:
		return tabBar + "\n" + m.settings.View()
	}
	return tabBar
}

// renderTabBar renders tab labels with the active tab highlighted.
func (m AppModel) renderTabBar() string {
	inactiveStyle := tabBarInactiveStyle()
	activeStyle := tabBarActiveStyle()

	labels := []string{" Markets ", " Portfolio ", " Settings "}
	rendered := make([]string, len(labels))
	for i, label := range labels {
		if m.activeTab == tab(i) {
			rendered[i] = activeStyle.Render(label)
		} else {
			rendered[i] = inactiveStyle.Render(label)
		}
	}

	return strings.Join(rendered, "  ")
}

// activeInputActive returns whether the currently active child has a text input focused.
func (m AppModel) activeInputActive() bool {
	switch m.activeTab {
	case tabMarkets:
		return m.markets.InputActive()
	case tabPortfolio:
		return m.portfolio.InputActive()
	case tabSettings:
		return m.settings.InputActive()
	}
	return false
}
