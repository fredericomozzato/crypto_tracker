package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	statusBarGray   = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	statusBarRed    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))
	statusBarGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	statusBarYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))
	statusBarOrange = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C00"))
)

// renderStatusBar renders a full-width status bar with left-aligned keyboard
// hints (rendered in gray) and an optional pre-styled right-aligned section.
// The right parameter must be pre-styled by the caller.
func renderStatusBar(width int, left, right string) string {
	leftStyled := statusBarGray.Render(left)
	padding := width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 1 {
		padding = 1
	}
	return leftStyled + strings.Repeat(" ", padding) + right
}
