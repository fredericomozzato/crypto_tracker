package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// PortfolioModel is the Portfolio tab. Slice 5: empty state only.
type PortfolioModel struct {
	width  int
	height int
}

// NewPortfolioModel creates a new PortfolioModel with zero values.
func NewPortfolioModel() PortfolioModel {
	return PortfolioModel{}
}

// update handles tea.WindowSizeMsg; ignores all other messages.
func (m PortfolioModel) update(msg tea.Msg) (PortfolioModel, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
	}
	return m, nil
}

// InputActive always returns false in this slice (no dialogs yet).
func (m PortfolioModel) InputActive() bool {
	return false
}

// View renders the empty-state message.
func (m PortfolioModel) View() string {
	return "no portfolios — press n to create one"
}
