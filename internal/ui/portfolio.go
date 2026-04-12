package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

// portfolioMode is the discriminated union for portfolio modes.
type portfolioMode interface{ isPortfolioMode() }

type (
	browsing struct{}
	creating struct{ input textinput.Model }
)

func (browsing) isPortfolioMode() {}
func (creating) isPortfolioMode() {}

// portfoliosLoadedMsg is sent when portfolios are loaded from the store.
// focusID is non-zero when the cursor should be positioned on a specific portfolio.
type portfoliosLoadedMsg struct {
	portfolios []store.Portfolio
	focusID    int64
}

// PortfolioModel is the Portfolio tab with two-panel layout.
type PortfolioModel struct {
	ctx        context.Context
	store      store.Store
	width      int
	height     int
	portfolios []store.Portfolio
	cursor     int
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
						return m, nil
					case 'k', 'K':
						m.moveCursor(-1)
						return m, nil
					case 'n', 'N':
						return m.openCreateDialog(), nil
					}
				}
			case tea.KeyDown:
				m.moveCursor(1)
				return m, nil
			case tea.KeyUp:
				m.moveCursor(-1)
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
		return m, nil

	case errMsg:
		m.lastErr = msg.err.Error()
		return m, nil
	}

	return m, nil
}

// InputActive returns true when a text input is focused.
func (m PortfolioModel) InputActive() bool {
	_, ok := m.mode.(creating)
	return ok
}

// View renders the two-panel layout with optional dialog overlay.
func (m PortfolioModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// When creating, render dialog centered instead of panels
	if mode, ok := m.mode.(creating); ok {
		dialog := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Render("New Portfolio\n\n" + mode.input.View())
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
	leftContent := m.renderLeftPanel(contentHeight-2, leftPanelInner) // -2 for border

	// Build right panel (placeholder)
	rightContent := m.renderRightPanel()

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

func (m PortfolioModel) renderRightPanel() string {
	titleStyle := lipgloss.NewStyle().Bold(true)
	title := titleStyle.Render("Holdings")

	if len(m.portfolios) == 0 {
		return title + "\n"
	}
	return title + "\n" + "no holdings"
}

func (m PortfolioModel) renderStatusBar() string {
	var content string
	switch m.mode.(type) {
	case browsing:
		content = "j/k portfolios • n new portfolio • q quit"
	case creating:
		content = "Enter to create • Esc to cancel"
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
