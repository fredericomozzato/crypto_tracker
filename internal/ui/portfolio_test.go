package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewPortfolioModel(t *testing.T) {
	m := NewPortfolioModel()
	if m.width != 0 || m.height != 0 {
		t.Errorf("expected zero-value model, got width=%d, height=%d", m.width, m.height)
	}
}

func TestPortfolioViewShowsEmptyState(t *testing.T) {
	m := NewPortfolioModel()
	m.width = 100
	m.height = 30

	view := m.View()
	if !strings.Contains(view, "no portfolios") {
		t.Errorf("expected view to contain 'no portfolios', got %q", view)
	}
}

func TestPortfolioViewShowsCreateHint(t *testing.T) {
	m := NewPortfolioModel()
	m.width = 100
	m.height = 30

	view := m.View()
	if !strings.Contains(view, "press n to create") {
		t.Errorf("expected view to contain 'press n to create', got %q", view)
	}
}

func TestPortfolioInputActiveFalse(t *testing.T) {
	m := NewPortfolioModel()
	if m.InputActive() {
		t.Error("expected InputActive() to return false for empty portfolio model")
	}
}

func TestPortfolioHandlesWindowSizeMsg(t *testing.T) {
	m := NewPortfolioModel()
	updated, _ := m.update(tea.WindowSizeMsg{Width: 120, Height: 39})

	if updated.width != 120 {
		t.Errorf("expected width 120, got %d", updated.width)
	}
	if updated.height != 39 {
		t.Errorf("expected height 39, got %d", updated.height)
	}
}

func TestPortfolioUpdateIgnoresOtherMessages(t *testing.T) {
	m := NewPortfolioModel()
	m.width = 100
	m.height = 30

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updated, cmd := m.update(msg)

	if cmd != nil {
		t.Errorf("expected nil cmd for arbitrary key, got %v", cmd)
	}
	if updated.width != 100 || updated.height != 30 {
		t.Error("expected model dimensions to be unchanged")
	}
}
