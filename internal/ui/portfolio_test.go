package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
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
	// Selected portfolio name should be present
	if !strings.Contains(view, "Portfolio A") {
		t.Error("expected view to contain selected portfolio name")
	}
	// Non-selected portfolio should also be present
	if !strings.Contains(view, "Portfolio B") {
		t.Error("expected view to contain non-selected portfolio name")
	}
	// No longer using ▶ prefix — highlight is via lipgloss.Reverse
	if strings.Contains(view, "▶") {
		t.Error("expected no ▶ cursor indicator (using reverse-video highlight instead)")
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

func TestPortfolioAKeyInBrowsingModeIsNoOp(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Error("expected nil cmd when pressing 'a' in browsing mode")
	}
}

func TestPortfolioAKeyInListModeOpensCoinPicker(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios:  []store.Portfolio{{ID: 1, Name: "Test"}},
		holdingRows: []store.HoldingRow{{ID: 1, CoinID: 1, Name: "Bitcoin", Ticker: "BTC"}},
	})
	m.width = 100
	m.height = 30
	// Load portfolios and holdings into model, then enter list mode
	m, _ = m.update(portfoliosLoadedMsg{portfolios: []store.Portfolio{{ID: 1, Name: "Test"}}})
	m, _ = m.update(holdingsLoadedMsg{holdings: []store.HoldingRow{{ID: 1, CoinID: 1, Name: "Bitcoin", Ticker: "BTC"}}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter}) // Enter list mode
	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Error("expected non-nil cmd when pressing 'a' in list mode")
	}
}

func TestCoinPickerReadyMsgEntersAddCoinMode(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	coins := threeCoins()
	updated, _ := m.update(coinPickerReadyMsg{coins: coins})
	if !updated.InputActive() {
		t.Error("expected InputActive() to be true after coinPickerReadyMsg")
	}
}

func TestCoinPickerReadyMsgWithNoCoinsShowsError(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	updated, _ := m.update(coinPickerReadyMsg{coins: nil})
	if updated.InputActive() {
		t.Error("expected InputActive() to be false when no coins available")
	}
	if updated.lastErr == "" {
		t.Error("expected error message when no coins available")
	}
}

func TestCoinPickerFiltersOutAlreadyHeldCoins(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	// First load holdings - coin ID 1 is held
	m, _ = m.update(holdingsLoadedMsg{holdings: []store.HoldingRow{{CoinID: 1, Name: "Coin 1"}}})
	// All threeCoins have IDs 1, 2, 3... coin ID 1 should be filtered out
	coins := threeCoins()
	updated, _ := m.update(coinPickerReadyMsg{coins: coins})
	// Check that we're in addCoin mode and only 2 coins are available
	if !updated.InputActive() {
		t.Fatal("expected to be in addCoin mode")
	}
	// Verify filtering: coin ID 1 should be filtered out, leaving 2 coins
	addCoinMode, ok := updated.mode.(addCoin)
	if !ok {
		t.Fatal("expected mode to be addCoin")
	}
	if len(addCoinMode.allCoins) != 2 {
		t.Errorf("expected 2 available coins after filtering, got %d", len(addCoinMode.allCoins))
	}
}

func TestCoinPickerAllHeldShowsError(t *testing.T) {
	coins := threeCoins()
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
		// Mark all coins as held
		holdingRows: []store.HoldingRow{
			{CoinID: coins[0].ID},
			{CoinID: coins[1].ID},
			{CoinID: coins[2].ID},
		},
	})
	m.width = 100
	m.height = 30
	m.holdings = []store.HoldingRow{
		{CoinID: coins[0].ID},
		{CoinID: coins[1].ID},
		{CoinID: coins[2].ID},
	}
	updated, _ := m.update(coinPickerReadyMsg{coins: coins})
	if updated.InputActive() {
		t.Error("expected InputActive() to be false when all coins are held")
	}
	if updated.lastErr == "" {
		t.Error("expected error message when all coins are held")
	}
}

func TestCoinPickerEscReturnsToBrowsing(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	coins := threeCoins()
	m, _ = m.update(coinPickerReadyMsg{coins: coins})
	if !m.InputActive() {
		t.Fatal("expected to be in addCoin mode")
	}
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEsc})
	if updated.InputActive() {
		t.Error("expected Esc to return to browsing mode")
	}
}

func TestCoinPickerJKNavigatesCursor(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	coins := threeCoins()
	m, _ = m.update(coinPickerReadyMsg{coins: coins})

	// Navigate down twice
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Get cursor position from addCoin mode
	if mode, ok := m.mode.(addCoin); ok {
		if mode.cursor != 2 {
			t.Errorf("expected cursor=2 after two 'j' presses, got %d", mode.cursor)
		}
	} else {
		t.Fatal("expected to be in addCoin mode")
	}
}

func TestCoinPickerCursorClampsAtTop(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	coins := threeCoins()
	m, _ = m.update(coinPickerReadyMsg{coins: coins})

	// k at top
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	if mode, ok := updated.mode.(addCoin); ok {
		if mode.cursor != 0 {
			t.Errorf("expected cursor to stay at 0, got %d", mode.cursor)
		}
	} else {
		t.Fatal("expected to be in addCoin mode")
	}
}

func TestCoinPickerCursorClampsAtBottom(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	coins := threeCoins()
	m, _ = m.update(coinPickerReadyMsg{coins: coins})

	// j to bottom and one more
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	if mode, ok := updated.mode.(addCoin); ok {
		if mode.cursor != 2 {
			t.Errorf("expected cursor to stay at 2, got %d", mode.cursor)
		}
	} else {
		t.Fatal("expected to be in addCoin mode")
	}
}

func TestCoinPickerEnterTransitionsToAddAmount(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	coins := threeCoins()
	m, _ = m.update(coinPickerReadyMsg{coins: coins})

	// Enter to select coin
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should now be in addAmount mode
	if _, ok := updated.mode.(addCoin); ok {
		t.Error("expected to leave addCoin mode after Enter")
	}
	if !updated.InputActive() {
		t.Error("expected InputActive() to still be true in addAmount mode")
	}
}

func TestAddAmountEscReturnsToCoinPicker(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	coins := threeCoins()
	m, _ = m.update(coinPickerReadyMsg{coins: coins})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter})

	// Esc should return to addCoin
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEsc})
	if _, ok := updated.mode.(addCoin); !ok {
		t.Error("expected Esc to return to addCoin mode")
	}
}

func TestAddAmountEnterWithEmptyIsNoOp(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	coins := threeCoins()
	m, _ = m.update(coinPickerReadyMsg{coins: coins})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter})

	// Enter with empty input
	updated, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected nil cmd for empty amount")
	}
	if !updated.InputActive() {
		t.Error("expected to stay in addAmount mode with empty input")
	}
}

func TestAddAmountEnterWithNonNumericSetsInlineError(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	coins := threeCoins()
	m, _ = m.update(coinPickerReadyMsg{coins: coins})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter})

	// Type "abc"
	for _, r := range []rune{'a', 'b', 'c'} {
		m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEnter})

	if !updated.InputActive() {
		t.Error("expected to stay in addAmount mode with invalid input")
	}
	if mode, ok := updated.mode.(addAmount); ok {
		if mode.errMsg == "" {
			t.Error("expected error message for invalid amount")
		}
	} else {
		t.Fatal("expected to be in addAmount mode")
	}
}

func TestAddAmountEnterWithZeroOrNegativeSetsInlineError(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	coins := threeCoins()
	m, _ = m.update(coinPickerReadyMsg{coins: coins})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter})

	// Type "0"
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEnter})

	if mode, ok := updated.mode.(addAmount); ok {
		if mode.errMsg == "" {
			t.Error("expected error message for zero amount")
		}
	} else {
		t.Fatal("expected to stay in addAmount mode")
	}
}

func TestAddAmountEnterWithValidAmountReturnsCmd(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	// Load portfolios into model
	m, _ = m.update(portfoliosLoadedMsg{portfolios: []store.Portfolio{{ID: 1, Name: "Test"}}})
	coins := threeCoins()
	m, _ = m.update(coinPickerReadyMsg{coins: coins})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter})

	// Type "1.5"
	for _, r := range []rune{'1', '.', '5'} {
		m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected non-nil cmd with valid amount")
	}
}

func TestHoldingsSavedMsgReturnsToBrowsing(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	updated, _ := m.update(holdingsSavedMsg{holdings: []store.HoldingRow{}})
	if updated.InputActive() {
		t.Error("expected holdingsSavedMsg to return to browsing mode")
	}
}

func TestHoldingsSavedMsgUpdatesHoldings(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	rows := []store.HoldingRow{
		{ID: 1, Name: "Bitcoin", Amount: 1.5},
	}
	updated, _ := m.update(holdingsSavedMsg{holdings: rows})
	if len(updated.holdings) != 1 {
		t.Errorf("expected 1 holding, got %d", len(updated.holdings))
	}
}

func TestHoldingsLoadedMsgUpdatesHoldings(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	rows := []store.HoldingRow{
		{ID: 1, Name: "Bitcoin", Amount: 1.5},
	}
	updated, _ := m.update(holdingsLoadedMsg{holdings: rows})
	if len(updated.holdings) != 1 {
		t.Errorf("expected 1 holding, got %d", len(updated.holdings))
	}
}

func TestFilterCoinsEmptyQuery(t *testing.T) {
	coins := threeCoins()
	result := filterCoins(coins, "")
	if len(result) != len(coins) {
		t.Errorf("expected %d coins with empty query, got %d", len(coins), len(result))
	}
}

func TestFilterCoinsByName(t *testing.T) {
	coins := []store.Coin{
		{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC"},
		{ApiID: "ethereum", Name: "Ethereum", Ticker: "ETH"},
	}
	result := filterCoins(coins, "bit")
	if len(result) != 1 || result[0].Name != "Bitcoin" {
		t.Errorf("expected only Bitcoin, got %v", result)
	}
}

func TestFilterCoinsByTicker(t *testing.T) {
	coins := []store.Coin{
		{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC"},
		{ApiID: "ethereum", Name: "Ethereum", Ticker: "ETH"},
	}
	result := filterCoins(coins, "eth")
	if len(result) != 1 || result[0].Ticker != "ETH" {
		t.Errorf("expected only ETH, got %v", result)
	}
}

func TestFilterCoinsByApiID(t *testing.T) {
	coins := []store.Coin{
		{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC"},
		{ApiID: "ethereum", Name: "Ethereum", Ticker: "ETH"},
	}
	result := filterCoins(coins, "bitco")
	if len(result) != 1 || result[0].ApiID != "bitcoin" {
		t.Errorf("expected only bitcoin, got %v", result)
	}
}

func TestFilterCoinsNoMatch(t *testing.T) {
	coins := threeCoins()
	result := filterCoins(coins, "xyz")
	if result == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 coins, got %d", len(result))
	}
}

// Slice 8 tests - List mode, Edit, Delete

func TestEnterFromBrowsingToListingMode(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin", Ticker: "BTC"},
	}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should be in listing mode
	if _, ok := updated.mode.(listing); !ok {
		t.Errorf("expected listing mode, got %T", updated.mode)
	}
	if updated.holdingsCursor != 0 {
		t.Errorf("expected holdingsCursor=0, got %d", updated.holdingsCursor)
	}
}

func TestEnterFromBrowsingNoHoldingsEntersListMode(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should enter list mode even with no holdings
	if _, ok := updated.mode.(listing); !ok {
		t.Errorf("expected listing mode, got %T", updated.mode)
	}
}

func TestListingJkNavigation(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
		{ID: 2, Name: "Ethereum"},
		{ID: 3, Name: "Litecoin"},
	}
	m.mode = listing{}
	m.holdingsCursor = 0

	// j twice
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.holdingsCursor != 2 {
		t.Errorf("expected holdingsCursor=2 after two 'j' presses, got %d", m.holdingsCursor)
	}

	// k once
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.holdingsCursor != 1 {
		t.Errorf("expected holdingsCursor=1 after 'k' press, got %d", m.holdingsCursor)
	}
}

func TestListingGJumpsToTop(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
		{ID: 2, Name: "Ethereum"},
		{ID: 3, Name: "Litecoin"},
	}
	m.mode = listing{}
	m.holdingsCursor = 2

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if updated.holdingsCursor != 0 {
		t.Errorf("expected holdingsCursor=0 after 'g', got %d", updated.holdingsCursor)
	}
}

func TestListingGJumpsToBottom(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
		{ID: 2, Name: "Ethereum"},
		{ID: 3, Name: "Litecoin"},
	}
	m.mode = listing{}
	m.holdingsCursor = 0

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if updated.holdingsCursor != 2 {
		t.Errorf("expected holdingsCursor=2 after 'G', got %d", updated.holdingsCursor)
	}
}

func TestListingClampsAtTop(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
		{ID: 2, Name: "Ethereum"},
	}
	m.mode = listing{}
	m.holdingsCursor = 0

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if updated.holdingsCursor != 0 {
		t.Errorf("expected holdingsCursor to stay at 0, got %d", updated.holdingsCursor)
	}
}

func TestListingClampsAtBottom(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
		{ID: 2, Name: "Ethereum"},
	}
	m.mode = listing{}
	m.holdingsCursor = 1

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if updated.holdingsCursor != 1 {
		t.Errorf("expected holdingsCursor to stay at 1, got %d", updated.holdingsCursor)
	}
}

func TestListingEscReturnsToBrowsing(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
	}
	m.mode = listing{}
	m.holdingsCursor = 0

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEsc})
	if _, ok := updated.mode.(browsing); !ok {
		t.Errorf("expected browsing mode after Esc, got %T", updated.mode)
	}
}

func TestListingEnterOpensEditDialog(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin", Ticker: "BTC", Amount: 1.5},
	}
	m.mode = listing{}
	m.holdingsCursor = 0

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	if _, ok := updated.mode.(editingAmount); !ok {
		t.Errorf("expected editingAmount mode, got %T", updated.mode)
	}
}

func TestListingXOpensDeleteDialog(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin", Ticker: "BTC", Amount: 1.5},
	}
	m.mode = listing{}
	m.holdingsCursor = 0

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	if _, ok := updated.mode.(deleting); !ok {
		t.Errorf("expected deleting mode, got %T", updated.mode)
	}
}

func TestListingAOpensCoinPicker(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin", Ticker: "BTC", Amount: 1.5},
	}
	m.mode = listing{}

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Error("expected non-nil cmd when pressing 'a' from list mode")
	}
}

func TestEditAmountEscReturnsToListing(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.mode = editingAmount{
		holding:  store.HoldingRow{ID: 1, Name: "Bitcoin"},
		input:    textinput.New(),
		listMode: listing{},
	}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEsc})
	if _, ok := updated.mode.(listing); !ok {
		t.Errorf("expected listing mode after Esc, got %T", updated.mode)
	}
}

func TestEditAmountEnterWithEmptyNoOp(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.mode = editingAmount{
		holding:  store.HoldingRow{ID: 1, Name: "Bitcoin"},
		input:    textinput.New(),
		listMode: listing{},
	}

	updated, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected nil cmd for empty input")
	}
	if _, ok := updated.mode.(editingAmount); !ok {
		t.Errorf("expected to stay in editingAmount mode, got %T", updated.mode)
	}
}

func TestEditAmountEnterWithNonNumericShowsError(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	ti := textinput.New()
	m.mode = editingAmount{
		holding:  store.HoldingRow{ID: 1, Name: "Bitcoin"},
		input:    ti,
		listMode: listing{},
	}

	// Type "abc"
	for _, r := range []rune{'a', 'b', 'c'} {
		m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEnter})

	if !updated.InputActive() {
		t.Error("expected to stay in editingAmount mode with invalid input")
	}
	if mode, ok := updated.mode.(editingAmount); ok {
		if mode.errMsg == "" {
			t.Error("expected error message for invalid amount")
		}
	} else {
		t.Fatal("expected to be in editingAmount mode")
	}
}

func TestEditAmountEnterWithZeroOrNegativeShowsError(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	ti := textinput.New()
	m.mode = editingAmount{
		holding:  store.HoldingRow{ID: 1, Name: "Bitcoin"},
		input:    ti,
		listMode: listing{},
	}

	// Type "0"
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEnter})

	if mode, ok := updated.mode.(editingAmount); ok {
		if mode.errMsg == "" {
			t.Error("expected error message for zero amount")
		}
	} else {
		t.Fatal("expected to stay in editingAmount mode")
	}
}

func TestEditAmountEnterWithValidAmountReturnsCmd(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	ti := textinput.New()
	ti.Focus() // Need to focus the input for it to receive keystrokes
	m.mode = editingAmount{
		holding:  store.HoldingRow{ID: 1, Name: "Bitcoin", CoinID: 1},
		input:    ti,
		listMode: listing{},
	}

	// Type "1.5"
	for _, r := range []rune{'1', '.', '5'} {
		m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected non-nil cmd with valid amount")
	}
}

func TestEditingAmountInputActive(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.mode = editingAmount{}
	if !m.InputActive() {
		t.Error("expected InputActive() to be true for editingAmount mode")
	}
}

func TestDeleteConfirmEscReturnsToListing(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.mode = deleting{
		holding:  store.HoldingRow{ID: 1, Name: "Bitcoin"},
		listMode: listing{},
	}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEsc})
	if _, ok := updated.mode.(listing); !ok {
		t.Errorf("expected listing mode after Esc, got %T", updated.mode)
	}
}

func TestDeleteConfirmEnterReturnsCmd(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.mode = deleting{
		holding:  store.HoldingRow{ID: 1, Name: "Bitcoin"},
		listMode: listing{},
	}

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected non-nil cmd when confirming delete")
	}
}

func TestDeletingInputActive(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.mode = deleting{}
	if m.InputActive() {
		t.Error("expected InputActive() to be false for deleting mode")
	}
}

func TestDeleteConfirmOtherKeysIgnored(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.mode = deleting{
		holding:  store.HoldingRow{ID: 1, Name: "Bitcoin"},
		listMode: listing{},
	}

	// Press various keys
	for _, key := range []rune{'j', 'k', 'a', 'x', '1'} {
		updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		if _, ok := updated.mode.(deleting); !ok {
			t.Errorf("expected to stay in deleting mode after pressing %c, got %T", key, updated.mode)
		}
	}
}

func TestCursorClampedAfterDelete(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
	}
	m.mode = listing{}
	m.holdingsCursor = 0 // at last item

	// Delete the last (and only) holding
	updated, _ := m.update(holdingDeletedMsg{holdings: []store.HoldingRow{}})
	if updated.holdingsCursor != 0 {
		t.Errorf("expected holdingsCursor=0 after delete, got %d", updated.holdingsCursor)
	}
}

func TestCursorStaysAtSamePositionAfterDelete(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
		{ID: 2, Name: "Ethereum"},
		{ID: 3, Name: "Litecoin"},
	}
	m.mode = listing{}
	m.holdingsCursor = 1 // on Ethereum

	// Delete Ethereum
	updated, _ := m.update(holdingDeletedMsg{holdings: []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
		{ID: 3, Name: "Litecoin"},
	}})
	if updated.holdingsCursor != 1 {
		t.Errorf("expected holdingsCursor=1 after delete, got %d", updated.holdingsCursor)
	}
}

func TestHoldingDeletedMsgUpdatesHoldings(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
		{ID: 2, Name: "Ethereum"},
	}

	updated, _ := m.update(holdingDeletedMsg{holdings: []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
	}})
	if len(updated.holdings) != 1 {
		t.Errorf("expected 1 holding after delete, got %d", len(updated.holdings))
	}
}

func TestHoldingDeletedMsgClampsCursor(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
		{ID: 2, Name: "Ethereum"},
	}
	m.holdingsCursor = 1

	updated, _ := m.update(holdingDeletedMsg{holdings: []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
	}})
	if updated.holdingsCursor != 0 {
		t.Errorf("expected holdingsCursor clamped to 0, got %d", updated.holdingsCursor)
	}
}

func TestHoldingDeletedMsgReturnsToListing(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
		{ID: 2, Name: "Ethereum"},
	}
	m.mode = deleting{}

	updated, _ := m.update(holdingDeletedMsg{holdings: []store.HoldingRow{
		{ID: 2, Name: "Ethereum"},
	}})
	if _, ok := updated.mode.(listing); !ok {
		t.Errorf("expected listing mode after delete, got %T", updated.mode)
	}
}

func TestHoldingDeletedMsgToBrowsingWhenEmpty(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
	}
	m.mode = deleting{}

	updated, _ := m.update(holdingDeletedMsg{holdings: []store.HoldingRow{}})
	if _, ok := updated.mode.(browsing); !ok {
		t.Errorf("expected browsing mode when holdings empty, got %T", updated.mode)
	}
}

func TestHoldingsSavedFromEditReturnsToListing(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
	}
	m.mode = editingAmount{}

	updated, _ := m.update(holdingsSavedMsg{holdings: []store.HoldingRow{
		{ID: 1, Name: "Bitcoin", Amount: 2.0},
	}})
	if _, ok := updated.mode.(listing); !ok {
		t.Errorf("expected listing mode after edit save, got %T", updated.mode)
	}
}

func TestHoldingsSavedFromAddReturnsToBrowsing(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.holdings = []store.HoldingRow{}
	m.mode = addAmount{}

	updated, _ := m.update(holdingsSavedMsg{holdings: []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
	}})
	if _, ok := updated.mode.(browsing); !ok {
		t.Errorf("expected browsing mode after add save, got %T", updated.mode)
	}
}

func TestBrowsingPgDnScrollsHoldingsPreview(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	// Create many holdings to allow scrolling
	for i := 0; i < 50; i++ {
		m.holdings = append(m.holdings, store.HoldingRow{ID: int64(i), Name: fmt.Sprintf("Coin%d", i)})
	}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyPgDown})
	if updated.scrollOffset <= 0 {
		t.Errorf("expected scrollOffset > 0 after PgDn, got %d", updated.scrollOffset)
	}
}

func TestBrowsingPgUpScrollsHoldingsPreview(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	// Create many holdings to allow scrolling
	for i := 0; i < 50; i++ {
		m.holdings = append(m.holdings, store.HoldingRow{ID: int64(i), Name: fmt.Sprintf("Coin%d", i)})
	}
	m.scrollOffset = 20

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyPgUp})
	if updated.scrollOffset >= 20 {
		t.Errorf("expected scrollOffset < 20 after PgUp, got %d", updated.scrollOffset)
	}
}

func TestBrowsingPgUpDoesNotGoBelowZero(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
	}
	m.scrollOffset = 0

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyPgUp})
	if updated.scrollOffset != 0 {
		t.Errorf("expected scrollOffset to stay at 0, got %d", updated.scrollOffset)
	}
}

func TestBrowsingJkResetsScrollOffset(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "Portfolio A"},
		{ID: 2, Name: "Portfolio B"},
	}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
	}
	m.scrollOffset = 10

	// j should reset scroll offset
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if updated.scrollOffset != 0 {
		t.Errorf("expected scrollOffset reset to 0 after j, got %d", updated.scrollOffset)
	}
}

func TestListingModeShowsPanelFocus(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
	}
	m.mode = listing{}

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in listing mode")
	}
}

func TestBrowsingModeShowsPanelFocus(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{ID: 1, Name: "Bitcoin"},
	}
	m.mode = browsing{}

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in browsing mode")
	}
}

func TestEditDialogShowsCoinName(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	ti := textinput.New()
	m.mode = editingAmount{
		holding: store.HoldingRow{ID: 1, Name: "Bitcoin", Ticker: "BTC", Amount: 1.5},
		input:   ti,
	}

	view := m.View()
	if !strings.Contains(view, "Bitcoin") {
		t.Error("expected edit dialog to show coin name")
	}
	if !strings.Contains(view, "BTC") {
		t.Error("expected edit dialog to show ticker")
	}
}

func TestDeleteDialogShowsCoinName(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.mode = deleting{
		holding: store.HoldingRow{ID: 1, Name: "Bitcoin", Ticker: "BTC", Amount: 1.5, Value: 50000},
	}

	view := m.View()
	if !strings.Contains(view, "Bitcoin") {
		t.Error("expected delete dialog to show coin name")
	}
	if !strings.Contains(view, "BTC") {
		t.Error("expected delete dialog to show ticker")
	}
	if !strings.Contains(view, "1.5000") && !strings.Contains(view, "1.5") {
		t.Error("expected delete dialog to show amount")
	}
}

func TestPortfolioViewShowsHoldingsTable(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{
		{Name: "Bitcoin", Value: 1000.0},
	}
	view := m.View()
	if !strings.Contains(view, "Bitcoin") {
		t.Error("expected view to contain 'Bitcoin'")
	}
}

func TestPortfolioViewShowsNoHoldingsMessage(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{{ID: 1, Name: "Test"}}
	m.holdings = []store.HoldingRow{}
	view := m.View()
	if !strings.Contains(view, "no holdings") {
		t.Errorf("expected view to contain 'no holdings', got: %s", view)
	}
}

func TestPortfolioInputActiveForAddCoinMode(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.mode = addCoin{}
	if !m.InputActive() {
		t.Error("expected InputActive() to be true for addCoin mode")
	}
}

func TestPortfolioInputActiveForAddAmountMode(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.mode = addAmount{}
	if !m.InputActive() {
		t.Error("expected InputActive() to be true for addAmount mode")
	}
}

// Slice 9 tests - Portfolio Edit and Delete

func TestBrowsingEKeyOpensEditDialog(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "Test Portfolio"},
	}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	if _, ok := updated.mode.(editingPortfolio); !ok {
		t.Errorf("expected editingPortfolio mode, got %T", updated.mode)
	}
}

func TestBrowsingEKeyNoPortfoliosIsNoOp(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	if _, ok := updated.mode.(browsing); !ok {
		t.Errorf("expected browsing mode when no portfolios, got %T", updated.mode)
	}
}

func TestEditingPortfolioEscReturnsToBrowsing(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.mode = editingPortfolio{
		portfolio: store.Portfolio{ID: 1, Name: "Test"},
		input:     textinput.New(),
	}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEsc})

	if _, ok := updated.mode.(browsing); !ok {
		t.Errorf("expected browsing mode after Esc, got %T", updated.mode)
	}
}

func TestEditingPortfolioEnterWithEmptyNameReturnsToBrowsing(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	ti := textinput.New()
	m.mode = editingPortfolio{
		portfolio: store.Portfolio{ID: 1, Name: "Test"},
		input:     ti,
	}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEnter})

	if _, ok := updated.mode.(browsing); !ok {
		t.Errorf("expected browsing mode after Enter with empty name, got %T", updated.mode)
	}
}

func TestEditingPortfolioEnterWithSameNameReturnsToBrowsing(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	ti := textinput.New()
	ti.SetValue("Test")
	m.mode = editingPortfolio{
		portfolio: store.Portfolio{ID: 1, Name: "Test"},
		input:     ti,
	}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEnter})

	if _, ok := updated.mode.(browsing); !ok {
		t.Errorf("expected browsing mode after Enter with same name, got %T", updated.mode)
	}
}

func TestEditingPortfolioEnterWithDuplicateNameShowsError(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "Portfolio A"},
		{ID: 2, Name: "Portfolio B"},
	}
	ti := textinput.New()
	ti.SetValue("Portfolio B") // Try to rename Portfolio A to Portfolio B
	m.mode = editingPortfolio{
		portfolio: store.Portfolio{ID: 1, Name: "Portfolio A"},
		input:     ti,
	}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEnter})

	if mode, ok := updated.mode.(editingPortfolio); ok {
		if mode.errMsg == "" {
			t.Error("expected error message for duplicate name")
		}
	} else {
		t.Errorf("expected to stay in editingPortfolio mode, got %T", updated.mode)
	}
}

func TestEditingPortfolioEnterWithValidNameReturnsCmd(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{
			{ID: 1, Name: "Old Name"},
		},
	})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "Old Name"},
	}
	ti := textinput.New()
	ti.Focus()
	ti.SetValue("New Name")
	m.mode = editingPortfolio{
		portfolio: store.Portfolio{ID: 1, Name: "Old Name"},
		input:     ti,
	}

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Error("expected non-nil cmd with valid name")
	}
}

func TestEditingPortfolioInputActive(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.mode = editingPortfolio{}

	if !m.InputActive() {
		t.Error("expected InputActive() to be true for editingPortfolio mode")
	}
}

func TestEditingPortfolioPrePopulatedName(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "Current Name"},
	}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	if mode, ok := updated.mode.(editingPortfolio); ok {
		if mode.input.Value() != "Current Name" {
			t.Errorf("expected input to be pre-populated with 'Current Name', got %q", mode.input.Value())
		}
	} else {
		t.Errorf("expected editingPortfolio mode, got %T", updated.mode)
	}
}

func TestBrowsingXKeyOpensDeleteDialog(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "Test Portfolio"},
	}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})

	if _, ok := updated.mode.(deletingPortfolio); !ok {
		t.Errorf("expected deletingPortfolio mode, got %T", updated.mode)
	}
}

func TestBrowsingXKeyNoPortfoliosIsNoOp(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})

	if _, ok := updated.mode.(browsing); !ok {
		t.Errorf("expected browsing mode when no portfolios, got %T", updated.mode)
	}
}

func TestDeletingPortfolioEscReturnsToBrowsing(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.mode = deletingPortfolio{
		portfolio: store.Portfolio{ID: 1, Name: "Test"},
	}

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyEsc})

	if _, ok := updated.mode.(browsing); !ok {
		t.Errorf("expected browsing mode after Esc, got %T", updated.mode)
	}
}

func TestDeletingPortfolioEnterReturnsCmd(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{
			{ID: 1, Name: "Test"},
		},
	})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "Test"},
	}
	m.mode = deletingPortfolio{
		portfolio: store.Portfolio{ID: 1, Name: "Test"},
	}

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Error("expected non-nil cmd when confirming delete")
	}
}

func TestDeletingPortfolioOtherKeysIgnored(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.mode = deletingPortfolio{
		portfolio: store.Portfolio{ID: 1, Name: "Test"},
	}

	// Press various keys
	for _, key := range []rune{'j', 'k', 'a', 'n', '1'} {
		updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		if _, ok := updated.mode.(deletingPortfolio); !ok {
			t.Errorf("expected to stay in deletingPortfolio mode after pressing %c, got %T", key, updated.mode)
		}
	}
}

func TestDeletingPortfolioInputActive(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.mode = deletingPortfolio{}

	if m.InputActive() {
		t.Error("expected InputActive() to be false for deletingPortfolio mode")
	}
}

func TestPortfolioDeletedMsgUpdatesPortfolios(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "Portfolio A"},
		{ID: 2, Name: "Portfolio B"},
	}

	updatedPortfolios := []store.Portfolio{
		{ID: 1, Name: "Portfolio A"},
	}

	updated, _ := m.update(portfolioDeletedMsg{portfolios: updatedPortfolios})

	if len(updated.portfolios) != 1 {
		t.Errorf("expected 1 portfolio after delete, got %d", len(updated.portfolios))
	}
}

func TestPortfolioDeletedMsgClampsCursor(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "Portfolio A"},
		{ID: 2, Name: "Portfolio B"},
		{ID: 3, Name: "Portfolio C"},
	}
	m.cursor = 2 // on last portfolio

	// Delete the last two portfolios
	updatedPortfolios := []store.Portfolio{
		{ID: 1, Name: "Portfolio A"},
	}

	updated, _ := m.update(portfolioDeletedMsg{portfolios: updatedPortfolios})

	if updated.cursor != 0 {
		t.Errorf("expected cursor clamped to 0, got %d", updated.cursor)
	}
}

func TestPortfolioDeletedMsgResetsScrollOffset(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "Portfolio A"},
	}
	m.scrollOffset = 10

	updated, _ := m.update(portfolioDeletedMsg{portfolios: []store.Portfolio{}})

	if updated.scrollOffset != 0 {
		t.Errorf("expected scrollOffset reset to 0, got %d", updated.scrollOffset)
	}
}

func TestPortfolioDeletedMsgToBrowsingWhenEmpty(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "Portfolio A"},
	}
	m.mode = deletingPortfolio{}

	updated, _ := m.update(portfolioDeletedMsg{portfolios: []store.Portfolio{}})

	if _, ok := updated.mode.(browsing); !ok {
		t.Errorf("expected browsing mode when all portfolios deleted, got %T", updated.mode)
	}

	if updated.holdings != nil {
		t.Error("expected holdings to be nil when no portfolios")
	}
}

func TestPortfolioDeletedMsgLoadsHoldingsForNewSelection(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{
			{ID: 1, Name: "Remaining"},
		},
	})
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "To Delete"},
		{ID: 2, Name: "Remaining"},
	}
	m.cursor = 0
	m.mode = deletingPortfolio{}

	updatedPortfolios := []store.Portfolio{
		{ID: 2, Name: "Remaining"},
	}

	_, cmd := m.update(portfolioDeletedMsg{portfolios: updatedPortfolios})

	if cmd == nil {
		t.Error("expected non-nil cmd to load holdings for new selection")
	}
}

func TestEditPortfolioDialogShowsCurrentName(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "Test Portfolio"},
	}
	ti := textinput.New()
	ti.SetValue("Test Portfolio")
	m.mode = editingPortfolio{
		portfolio: store.Portfolio{ID: 1, Name: "Test Portfolio"},
		input:     ti,
	}

	view := m.View()
	if !strings.Contains(view, "Test Portfolio") {
		t.Error("expected edit dialog to show portfolio name")
	}
}

func TestDeletePortfolioDialogShowsName(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	m.portfolios = []store.Portfolio{
		{ID: 1, Name: "Test Portfolio"},
	}
	m.mode = deletingPortfolio{
		portfolio: store.Portfolio{ID: 1, Name: "Test Portfolio"},
	}

	view := m.View()
	if !strings.Contains(view, "Test Portfolio") {
		t.Error("expected delete dialog to show portfolio name")
	}
}

func TestCoinPickerTypingFilters(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	coins := []store.Coin{
		{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC"},
		{ApiID: "ethereum", Name: "Ethereum", Ticker: "ETH"},
		{ApiID: "litecoin", Name: "Litecoin", Ticker: "LTC"},
	}
	m, _ = m.update(coinPickerReadyMsg{coins: coins})

	// Type "bit" to filter
	for _, r := range []rune{'b', 'i', 't'} {
		m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	if mode, ok := m.mode.(addCoin); ok {
		if len(mode.filtered) != 1 || mode.filtered[0].Name != "Bitcoin" {
			t.Errorf("expected only Bitcoin after typing 'bit', got %v", mode.filtered)
		}
	} else {
		t.Fatal("expected to be in addCoin mode")
	}
}

func TestCoinPickerCursorClampedAfterFilter(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	coins := []store.Coin{
		{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC"},
		{ApiID: "ethereum", Name: "Ethereum", Ticker: "ETH"},
		{ApiID: "litecoin", Name: "Litecoin", Ticker: "LTC"},
	}
	m, _ = m.update(coinPickerReadyMsg{coins: coins})

	// Move cursor to position 2 (Litecoin)
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Verify cursor is at 2
	if mode, ok := m.mode.(addCoin); ok {
		if mode.cursor != 2 {
			t.Fatalf("expected cursor=2, got %d", mode.cursor)
		}
	} else {
		t.Fatal("expected to be in addCoin mode")
	}

	// Type "bit" to filter down to 1 result
	for _, r := range []rune{'b', 'i', 't'} {
		m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Cursor should be clamped to 0 since there's only 1 result
	if mode, ok := m.mode.(addCoin); ok {
		if mode.cursor != 0 {
			t.Errorf("expected cursor to be clamped to 0 after filter, got %d", mode.cursor)
		}
	} else {
		t.Fatal("expected to be in addCoin mode")
	}
}
