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

type browsing struct{}
type creating struct{ input textinput.Model }

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
		switch m.mode.(type) {
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
				name := strings.TrimSpace(m.mode.(creating).input.Value())
				if name != "" {
					return m, m.cmdCreatePortfolio(name)
				}
				return m, nil
			default:
				// Delegate to input
				mode := m.mode.(creating)
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

	contentHeight := m.height - 1 // Reserve 1 row for status bar

	// Build left panel
	leftWidth := 30
	leftContent := m.renderLeftPanel(contentHeight)

	// Build right panel (placeholder)
	rightWidth := m.width - leftWidth - 1 // -1 for separator
	rightContent := m.renderRightPanel(contentHeight)

	// Combine panels
	leftStyle := lipgloss.NewStyle().Width(leftWidth).Height(contentHeight)
	rightStyle := lipgloss.NewStyle().Width(rightWidth).Height(contentHeight)

	panels := lipgloss.JoinHorizontal(lipgloss.Top,
		leftStyle.Render(leftContent),
		" ",
		rightStyle.Render(rightContent),
	)

	// Status bar
	statusBar := m.renderStatusBar()

	view := panels + "\n" + statusBar

	// Overlay dialog if creating
	if _, ok := m.mode.(creating); ok {
		view = m.renderDialogOverlay(view)
	}

	return view
}

func (m PortfolioModel) renderLeftPanel(height int) string {
	if len(m.portfolios) == 0 {
		return "no portfolios — press n to create one"
	}

	var b strings.Builder
	for i, p := range m.portfolios {
		if i >= height {
			break
		}
		prefix := "  "
		if i == m.cursor {
			prefix = "▶ "
		}
		b.WriteString(prefix + p.Name + "\n")
	}
	return b.String()
}

func (m PortfolioModel) renderRightPanel(height int) string {
	if len(m.portfolios) == 0 {
		return ""
	}
	return "no holdings"
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

func (m PortfolioModel) renderDialogOverlay(background string) string {
	dialogWidth := 40
	dialogHeight := 5

	mode := m.mode.(creating)
	inputView := mode.input.View()

	content := "New Portfolio\n\n" + inputView

	dialog := lipgloss.NewStyle().
		Width(dialogWidth).
		Height(dialogHeight).
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Render(content)

	// Center the dialog over the background
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#000000")),
	)
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
