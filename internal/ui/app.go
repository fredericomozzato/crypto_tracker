package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AppModel is the root Bubble Tea model for the crypto-tracker TUI.
type AppModel struct {
	width  int
	height int
}

// NewAppModel creates a new AppModel with default values.
func NewAppModel() AppModel {
	return AppModel{}
}

// Init is the Bubble Tea init command. Returns nil (no initial command).
func (m AppModel) Init() tea.Cmd {
	return nil
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
	}

	return m, nil
}

// View renders the current state of the app.
func (m AppModel) View() string {
	const placeholder = "crypto-tracker — press q to quit"

	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)

	return style.Render(placeholder)
}
