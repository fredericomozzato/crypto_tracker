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

func TestPortfolioAKeyWhenNoPortfoliosIsNoOp(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{})
	m.width = 100
	m.height = 30
	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Error("expected nil cmd when pressing 'a' with no portfolios")
	}
}

func TestPortfolioAKeyOpensCoinPicker(t *testing.T) {
	m := NewPortfolioModel(testCtx, &StubStore{
		portfolios: []store.Portfolio{{ID: 1, Name: "Test"}},
	})
	m.width = 100
	m.height = 30
	// Load portfolios into model
	m, _ = m.update(portfoliosLoadedMsg{portfolios: []store.Portfolio{{ID: 1, Name: "Test"}}})
	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Error("expected non-nil cmd when pressing 'a' with portfolios")
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
