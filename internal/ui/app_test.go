package ui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

func TestNewAppModel(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)

	view := m.View()
	if view == "" {
		t.Error("expected non-empty View from NewAppModel, got empty string")
	}
}

func TestAppModelDefaultTabIsMarkets(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)

	if m.activeTab != tabMarkets {
		t.Errorf("expected activeTab to be tabMarkets (%d), got %d", tabMarkets, m.activeTab)
	}
}

func TestAppModelViewContainsTabBar(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30

	view := m.View()
	if !strings.Contains(view, "Markets") {
		t.Errorf("expected view to contain 'Markets', got %q", view)
	}
	if !strings.Contains(view, "Portfolio") {
		t.Errorf("expected view to contain 'Portfolio', got %q", view)
	}
}

func TestQuitOnQ(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Fatal("expected non-nil cmd when pressing q")
	}

	result := cmd()
	if _, ok := result.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", result)
	}
}

func TestQuitOnCtrlC(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Fatal("expected non-nil cmd when pressing ctrl+c")
	}

	result := cmd()
	if _, ok := result.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", result)
	}
}

func TestTabKeyAdvancesToPortfolio(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30

	if m.activeTab != tabMarkets {
		t.Fatalf("expected initial tab to be Markets, got %d", m.activeTab)
	}

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, _ := m.Update(msg)
	model := updated.(AppModel)

	if model.activeTab != tabPortfolio {
		t.Errorf("expected Tab to switch to Portfolio (%d), got %d", tabPortfolio, model.activeTab)
	}
}

func TestTabKeyWrapsToMarkets(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.activeTab = tabPortfolio
	m.width = 100
	m.height = 30

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, _ := m.Update(msg)
	model := updated.(AppModel)

	if model.activeTab != tabMarkets {
		t.Errorf("expected Tab to wrap to Markets (%d), got %d", tabMarkets, model.activeTab)
	}
}

func TestShiftTabGoesBackToMarkets(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.activeTab = tabPortfolio
	m.width = 100
	m.height = 30

	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	updated, _ := m.Update(msg)
	model := updated.(AppModel)

	if model.activeTab != tabMarkets {
		t.Errorf("expected Shift+Tab to go back to Markets (%d), got %d", tabMarkets, model.activeTab)
	}
}

func TestShiftTabWrapsToPortfolio(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.activeTab = tabMarkets
	m.width = 100
	m.height = 30

	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	updated, _ := m.Update(msg)
	model := updated.(AppModel)

	if model.activeTab != tabPortfolio {
		t.Errorf("expected Shift+Tab to wrap to Portfolio (%d), got %d", tabPortfolio, model.activeTab)
	}
}

func TestOneKeySelectsMarkets(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.activeTab = tabPortfolio
	m.width = 100
	m.height = 30

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
	updated, _ := m.Update(msg)
	model := updated.(AppModel)

	if model.activeTab != tabMarkets {
		t.Errorf("expected '1' to select Markets (%d), got %d", tabMarkets, model.activeTab)
	}
}

func TestTwoKeySelectsPortfolio(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
	updated, _ := m.Update(msg)
	model := updated.(AppModel)

	if model.activeTab != tabPortfolio {
		t.Errorf("expected '2' to select Portfolio (%d), got %d", tabPortfolio, model.activeTab)
	}
}

func TestCtrlCQuitsFromPortfolioTab(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.activeTab = tabPortfolio
	m.width = 100
	m.height = 30

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Fatal("expected non-nil cmd when pressing Ctrl+C from Portfolio tab")
	}

	result := cmd()
	if _, ok := result.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", result)
	}
}

func TestQuitFromPortfolioTab(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.activeTab = tabPortfolio
	m.width = 100
	m.height = 30

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Fatal("expected non-nil cmd when pressing 'q' from Portfolio tab")
	}

	result := cmd()
	if _, ok := result.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", result)
	}
}

func TestWindowSizeMsgSetsRootDimensions(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)
	model := updated.(AppModel)

	if model.width != 120 {
		t.Errorf("expected width 120, got %d", model.width)
	}
	if model.height != 40 {
		t.Errorf("expected height 40, got %d", model.height)
	}
}

func TestWindowSizeMsgPropagatesAdjustedHeightToChildren(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)
	model := updated.(AppModel)

	if model.markets.height != 39 {
		t.Errorf("expected markets.height 39 (40-1 for tab bar), got %d", model.markets.height)
	}
	if model.portfolio.height != 39 {
		t.Errorf("expected portfolio.height 39 (40-1 for tab bar), got %d", model.portfolio.height)
	}
}

func TestInitReturnsBatchedCmd(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	cmd := m.Init()

	if cmd == nil {
		t.Fatal("expected non-nil cmd from Init, got nil")
	}
}

func TestActiveInputActiveFalse(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)

	if m.activeInputActive() {
		t.Error("expected activeInputActive() to return false by default")
	}
}

func TestViewShowsPortfolioEmptyState(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.activeTab = tabPortfolio
	// Set dimensions via WindowSizeMsg to propagate to children
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(AppModel)

	view := m.View()
	if !strings.Contains(view, "no portfolios") {
		t.Errorf("expected view to contain 'no portfolios' when on Portfolio tab, got %q", view)
	}
}

func TestBackgroundMessagesFannedOutToBothChildren(t *testing.T) {
	// This test verifies the fix for I1: portfoliosLoadedMsg was being dropped
	// when Markets tab was active on startup because messages were only routed
	// to the active tab.
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	// Start on Markets tab (default)
	m.activeTab = tabMarkets
	m.width = 100
	m.height = 30

	// Set dimensions on children
	m.markets.width = 100
	m.markets.height = 29
	m.portfolio.width = 100
	m.portfolio.height = 29

	// Simulate portfoliosLoadedMsg arriving while Markets tab is active
	portfolios := []store.Portfolio{
		{ID: 1, Name: "Test Portfolio"},
	}
	msg := portfoliosLoadedMsg{portfolios: portfolios}
	updated, _ := m.Update(msg)
	model := updated.(AppModel)

	// Portfolio model should have received the message even though
	// Markets tab was active
	if len(model.portfolio.portfolios) != 1 {
		t.Errorf("expected portfolio model to receive portfoliosLoadedMsg, got %d portfolios", len(model.portfolio.portfolios))
	}
}

func TestPanelTitlesDisplayed(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.activeTab = tabPortfolio
	// Set dimensions via WindowSizeMsg to propagate to children
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(AppModel)

	view := m.View()
	// Check for left panel title
	if !strings.Contains(view, "Portfolios") {
		t.Errorf("expected view to contain 'Portfolios' title, got %q", view)
	}
	// Check for right panel title
	if !strings.Contains(view, "Holdings") {
		t.Errorf("expected view to contain 'Holdings' title, got %q", view)
	}
}
