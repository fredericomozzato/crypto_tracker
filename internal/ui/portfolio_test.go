package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

func TestNewPortfolioModel(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	if m.width != 0 || m.height != 0 {
		t.Errorf("expected zero-value dimensions, got width=%d, height=%d", m.width, m.height)
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor=0, got %d", m.cursor)
	}
	if len(m.portfolios) != 0 {
		t.Errorf("expected 0 portfolios, got %d", len(m.portfolios))
	}
	// Should be in browsing mode
	if m.InputActive() {
		t.Error("expected InputActive() to be false for new model")
	}
}

func TestPortfolioInputActiveFalseWhenBrowsing(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	if m.InputActive() {
		t.Error("expected InputActive() to return false when browsing")
	}
}

func TestPortfolioInputActiveTrueWhenCreating(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !updated.InputActive() {
		t.Error("expected InputActive() to return true after pressing 'n'")
	}
}

func TestPortfolioNKeyOpensCreateDialog(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !updated.InputActive() {
		t.Error("expected 'n' key to switch to creating mode")
	}
}

func TestPortfolioCreateDialogEscCancels(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !m.InputActive() {
		t.Fatal("expected to be in creating mode")
	}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEsc})
	if updated.InputActive() {
		t.Error("expected Esc to return to browsing mode")
	}
}

func TestPortfolioCreateDialogEnterWithEmptyIsNoOp(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !m.InputActive() {
		t.Fatal("expected to be in creating mode")
	}

	updated, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected nil cmd for empty input")
	}
	if !updated.InputActive() {
		t.Error("expected to stay in creating mode with empty input")
	}
}

func TestPortfolioCreateDialogEnterWithTextReturnsCmd(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !m.InputActive() {
		t.Fatal("expected to be in creating mode")
	}

	// Type "My Portfolio"
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'M'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected non-nil cmd when Enter pressed with text")
	}
}

func TestPortfolioJKNavigation(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30

	// Load 3 portfolios
	threePortfolios := []store.Portfolio{
		{ID: 1, Name: "Portfolio A"},
		{ID: 2, Name: "Portfolio B"},
		{ID: 3, Name: "Portfolio C"},
	}
	m, _ = m.update(portfoliosLoadedMsg{portfolios: threePortfolios})

	// j twice
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 2 {
		t.Errorf("expected cursor=2 after two 'j' presses, got %d", m.cursor)
	}

	// k once
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.cursor != 1 {
		t.Errorf("expected cursor=1 after 'k' press, got %d", m.cursor)
	}
}

func TestPortfolioCursorClampsAtTop(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30

	threePortfolios := []store.Portfolio{
		{ID: 1, Name: "Portfolio A"},
		{ID: 2, Name: "Portfolio B"},
		{ID: 3, Name: "Portfolio C"},
	}
	m, _ = m.update(portfoliosLoadedMsg{portfolios: threePortfolios})

	// k at top
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.cursor != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", m.cursor)
	}
}

func TestPortfolioCursorClampsAtBottom(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30

	threePortfolios := []store.Portfolio{
		{ID: 1, Name: "Portfolio A"},
		{ID: 2, Name: "Portfolio B"},
		{ID: 3, Name: "Portfolio C"},
	}
	m, _ = m.update(portfoliosLoadedMsg{portfolios: threePortfolios})

	// Move to last
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 2 {
		t.Fatalf("expected cursor=2, got %d", m.cursor)
	}

	// j at bottom
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 2 {
		t.Errorf("expected cursor to stay at 2, got %d", m.cursor)
	}
}

func TestPortfoliosLoadedMsgPopulatesModel(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	threePortfolios := []store.Portfolio{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
		{ID: 3, Name: "C"},
	}
	updated, _ := m.update(portfoliosLoadedMsg{portfolios: threePortfolios})
	if len(updated.portfolios) != 3 {
		t.Errorf("expected 3 portfolios, got %d", len(updated.portfolios))
	}
}

func TestPortfoliosLoadedMsgPositionsCursorOnFocusID(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	threePortfolios := []store.Portfolio{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
		{ID: 3, Name: "C"},
	}
	updated, _ := m.update(portfoliosLoadedMsg{portfolios: threePortfolios, focusID: 3})
	if updated.cursor != 2 {
		t.Errorf("expected cursor=2 for focusID=3, got %d", updated.cursor)
	}
}

func TestPortfoliosLoadedMsgSwitchesToBrowsing(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !m.InputActive() {
		t.Fatal("expected to be in creating mode")
	}

	updated, _ := m.update(portfoliosLoadedMsg{portfolios: []store.Portfolio{{ID: 1, Name: "Test"}}})
	if updated.InputActive() {
		t.Error("expected portfoliosLoadedMsg to switch to browsing mode")
	}
}

func TestPortfolioViewShowsPortfolioNames(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "Long Term"},
		{ID: 2, Name: "Short Term"},
	}

	view := m.View()
	if !strings.Contains(view, "Long Term") {
		t.Error("expected view to contain 'Long Term'")
	}
	if !strings.Contains(view, "Short Term") {
		t.Error("expected view to contain 'Short Term'")
	}
}

func TestPortfolioViewShowsCursorIndicator(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "Portfolio A"},
		{ID: 2, Name: "Portfolio B"},
	}
	m.cursor = 0

	view := m.View()
	if !strings.Contains(view, "▶") {
		t.Error("expected view to contain cursor indicator '▶'")
	}
}

func TestPortfolioViewShowsEmptyStateWhenNoPortfolios(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30

	view := m.View()
	if !strings.Contains(view, "no portfolios") {
		t.Errorf("expected view to contain 'no portfolios', got %q", view)
	}
}

func TestPortfolioHandlesWindowSizeMsg(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	updated, _ := m.update(tea.WindowSizeMsg{Width: 120, Height: 39})

	if updated.width != 120 {
		t.Errorf("expected width 120, got %d", updated.width)
	}
	if updated.height != 39 {
		t.Errorf("expected height 39, got %d", updated.height)
	}
}

func threePortfolios() []store.Portfolio {
	return []store.Portfolio{
		{ID: 1, Name: "Portfolio A"},
		{ID: 2, Name: "Portfolio B"},
		{ID: 3, Name: "Portfolio C"},
	}
}
