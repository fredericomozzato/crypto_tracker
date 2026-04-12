package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

func TestMarketsInitReturnsBatchedCmd(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	cmd := m.Init()

	if cmd == nil {
		t.Fatal("expected non-nil cmd from Init, got nil")
	}
}

func TestMarketsCoinsLoadedMsg(t *testing.T) {
	s := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, s, api)
	m.width = 100
	m.height = 30

	coin := store.Coin{
		ApiID:       "bitcoin",
		Name:        "Bitcoin",
		Ticker:      "BTC",
		Rate:        67000.00,
		PriceChange: -1.23,
		MarketRank:  1,
	}

	coins := []store.Coin{coin}

	msg := coinsLoadedMsg{coins: coins}
	updated, _ := m.update(msg)

	view := updated.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}

	if !strings.Contains(view, "Bitcoin") {
		t.Errorf("expected view to contain 'Bitcoin', got %q", view)
	}

	if !strings.Contains(view, "BTC") {
		t.Errorf("expected view to contain 'BTC', got %q", view)
	}

	if !strings.Contains(view, "$67,000.00") {
		t.Errorf("expected view to contain '$67,000.00', got %q", view)
	}

	if !strings.Contains(view, "Name") {
		t.Errorf("expected view to contain column header 'Name', got %q", view)
	}
	if !strings.Contains(view, "Ticker") {
		t.Errorf("expected view to contain column header 'Ticker', got %q", view)
	}
	if !strings.Contains(view, "Price (USD)") {
		t.Errorf("expected view to contain column header 'Price (USD)', got %q", view)
	}
	if !strings.Contains(view, "24h") {
		t.Errorf("expected view to contain column header '24h', got %q", view)
	}
}

func TestMarketsErrMsg(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30

	testErr := "connection failed"
	msg := errMsg{err: errors.New(testErr)}
	updated, _ := m.update(msg)

	view := updated.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}

	if !strings.Contains(view, testErr) {
		t.Errorf("expected view to contain error %q, got %q", testErr, view)
	}
}

func TestMarketsViewRendersLoading(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}

	if !strings.Contains(view, "loading") {
		t.Errorf("expected view to contain 'loading', got %q", view)
	}
}

func TestMarketsViewRendersColumnHeaders(t *testing.T) {
	coins := threeCoins()
	m := NewMarketsModel(testCtx, &StubStore{coins: coins}, &StubAPI{})
	m.width = 120
	m.height = 40
	updated, _ := m.update(coinsLoadedMsg{coins: coins})

	view := updated.View()
	for _, col := range []string{"#", "Name", "Ticker", "Price (USD)", "24h"} {
		if !strings.Contains(view, col) {
			t.Errorf("expected view to contain header %q, got %q", col, view)
		}
	}
}

func TestMarketsViewRendersHintLine(t *testing.T) {
	coins := threeCoins()
	m := NewMarketsModel(testCtx, &StubStore{coins: coins}, &StubAPI{})
	m.width = 120
	m.height = 40
	updated, _ := m.update(coinsLoadedMsg{coins: coins})

	view := updated.View()
	if !strings.Contains(view, "j/k") {
		t.Errorf("expected view to contain 'j/k', got %q", view)
	}
}

func TestMarketsRefreshKeyReturnsCmdWhenCoinsLoaded(t *testing.T) {
	storeStub := &StubStore{coins: []store.Coin{{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC", Rate: 67000.00}}}
	api := &StubAPI{prices: map[string]float64{"bitcoin": 68000.00}}
	m := NewMarketsModel(testCtx, storeStub, api)
	m.width = 100
	m.height = 30

	updated, _ := m.update(coinsLoadedMsg{coins: storeStub.coins})
	m = updated

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	updated, cmd := m.update(msg)

	if !updated.refreshing {
		t.Error("expected refreshing to be true")
	}

	if cmd == nil {
		t.Fatal("expected non-nil cmd when pressing r with coins loaded")
	}
}

func TestMarketsRefreshKeyIgnoredWhenAlreadyRefreshing(t *testing.T) {
	stub := &StubStore{coins: []store.Coin{{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC", Rate: 67000.00}}}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30
	m.coins = stub.coins
	m.refreshing = true

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	_, cmd := m.update(msg)

	if cmd != nil {
		t.Error("expected nil cmd when already refreshing")
	}
}

func TestMarketsRefreshKeyIgnoredWhenNoCoins(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	_, cmd := m.update(msg)

	if cmd != nil {
		t.Error("expected nil cmd when no coins loaded")
	}
}

func TestMarketsPricesUpdatedMsg(t *testing.T) {
	storeStub := &StubStore{coins: []store.Coin{{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC", Rate: 68000.00}}}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, storeStub, api)
	m.width = 100
	m.height = 30
	m.coins = []store.Coin{{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC", Rate: 67000.00}}
	m.refreshing = true

	msg := pricesUpdatedMsg{coins: storeStub.coins}
	updated, _ := m.update(msg)

	if updated.refreshing {
		t.Error("expected refreshing to be false after pricesUpdatedMsg")
	}

	if len(updated.coins) != 1 || updated.coins[0].Rate != 68000.00 {
		t.Errorf("expected updated coin with Rate 68000.00, got %v", updated.coins)
	}
}

func TestMarketsViewShowsRefreshHint(t *testing.T) {
	stub := &StubStore{coins: []store.Coin{{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC", Rate: 67000.00, MarketRank: 1}}}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30
	m.coins = stub.coins

	view := m.View()
	if !strings.Contains(view, "r refresh") {
		t.Errorf("expected view to contain 'r refresh', got %q", view)
	}
}

func TestMarketsCursorMovesDownOnJ(t *testing.T) {
	m := setupMarketsModel(t, threeCoins())
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if updated.cursor != 1 {
		t.Errorf("expected cursor 1 after 'j', got %d", updated.cursor)
	}
}

func TestMarketsCursorMovesUpOnK(t *testing.T) {
	m := setupMarketsModel(t, threeCoins())
	m.cursor = 1
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if updated.cursor != 0 {
		t.Errorf("expected cursor 0 after 'k', got %d", updated.cursor)
	}
}

func TestMarketsCursorClampsAtBottom(t *testing.T) {
	m := setupMarketsModel(t, threeCoins())
	m.cursor = 2
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if updated.cursor != 2 {
		t.Errorf("expected cursor 2 (clamped), got %d", updated.cursor)
	}
}

func TestMarketsCursorClampsAtTop(t *testing.T) {
	m := setupMarketsModel(t, threeCoins())
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if updated.cursor != 0 {
		t.Errorf("expected cursor 0 (clamped), got %d", updated.cursor)
	}
}

func TestMarketsCursorJumpsToTopOnG(t *testing.T) {
	m := setupMarketsModel(t, threeCoins())
	m.cursor = 2
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if updated.cursor != 0 {
		t.Errorf("expected cursor 0 after 'g', got %d", updated.cursor)
	}
}

func TestMarketsCursorJumpsToBottomOnCapG(t *testing.T) {
	m := setupMarketsModel(t, threeCoins())
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if updated.cursor != 2 {
		t.Errorf("expected cursor 2 after 'G', got %d", updated.cursor)
	}
}

func TestMarketsCursorMovesDownOnDownArrow(t *testing.T) {
	m := setupMarketsModel(t, threeCoins())
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyDown})
	if updated.cursor != 1 {
		t.Errorf("expected cursor 1 after KeyDown, got %d", updated.cursor)
	}
}

func TestMarketsCursorMovesUpOnUpArrow(t *testing.T) {
	m := setupMarketsModel(t, threeCoins())
	m.cursor = 1
	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyUp})
	if updated.cursor != 0 {
		t.Errorf("expected cursor 0 after KeyUp, got %d", updated.cursor)
	}
}

func TestMarketsMoveCursorNoPanicOnEmptyCoins(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30

	updated, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if updated.cursor != 0 {
		t.Errorf("expected cursor 0 on empty coins, got %d", updated.cursor)
	}

	updated, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if updated.cursor != 0 {
		t.Errorf("expected cursor 0 on empty coins after 'k', got %d", updated.cursor)
	}
}

func TestMarketsCursorClampedAfterCoinsLoaded(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30

	m.cursor = 5
	updated, _ := m.update(coinsLoadedMsg{coins: threeCoins()})
	if updated.cursor != 2 {
		t.Errorf("expected cursor clamped to 2 (last index), got %d", updated.cursor)
	}
}

func TestMarketsTickMsgAlwaysReissuesTicker(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)

	updated, cmd := m.update(tickMsg(time.Now()))

	if cmd == nil {
		t.Error("expected non-nil cmd from tickMsg (ticker should be re-armed)")
	}
	_ = updated
}

func TestMarketsTickMsgBelow60sDoesNotRefresh(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.coins = threeCoins()
	m.lastRefreshed = time.Now().Add(-30 * time.Second)

	updated, _ := m.update(tickMsg(time.Now()))

	if updated.refreshing {
		t.Error("expected refreshing to stay false when 60s not elapsed")
	}
}

func TestMarketsTickMsgAbove60sFiresRefresh(t *testing.T) {
	stub := &StubStore{coins: threeCoins()}
	api := &StubAPI{prices: map[string]float64{"coin-1": 100.0}}
	m := NewMarketsModel(testCtx, stub, api)
	m.coins = threeCoins()
	m.lastRefreshed = time.Now().Add(-61 * time.Second)
	m.refreshing = false

	updated, cmd := m.update(tickMsg(time.Now()))

	if !updated.refreshing {
		t.Error("expected refreshing to be true when 60+ seconds elapsed")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd when refresh fires")
	}
}

func TestMarketsTickMsgWhenAlreadyRefreshing(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.coins = threeCoins()
	m.lastRefreshed = time.Now().Add(-61 * time.Second)
	m.refreshing = true

	updated, cmd := m.update(tickMsg(time.Now()))

	if !updated.refreshing {
		t.Error("expected refreshing to remain true")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (ticker re-arm) even when already refreshing")
	}
}

func TestMarketsTickMsgWhenNoCoins(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.lastRefreshed = time.Now().Add(-61 * time.Second)

	updated, _ := m.update(tickMsg(time.Now()))

	if updated.refreshing {
		t.Error("expected no refresh when no coins loaded")
	}
}

func TestMarketsCoinsLoadedSetsLastRefreshed(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30

	if !m.lastRefreshed.IsZero() {
		t.Error("expected lastRefreshed to be zero initially")
	}

	updated, _ := m.update(coinsLoadedMsg{coins: threeCoins()})

	if updated.lastRefreshed.IsZero() {
		t.Error("expected lastRefreshed to be set after coinsLoadedMsg")
	}
}

func TestMarketsPricesUpdatedSetsLastRefreshed(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.refreshing = true

	updated, _ := m.update(pricesUpdatedMsg{coins: threeCoins()})

	if updated.lastRefreshed.IsZero() {
		t.Error("expected lastRefreshed to be set after pricesUpdatedMsg")
	}
	if updated.refreshing {
		t.Error("expected refreshing to be false after pricesUpdatedMsg")
	}
}

func TestMarketsStatusBarShowsLoading(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30

	right := m.statusRight()
	if right != "loading..." {
		t.Errorf("expected statusRight 'loading...', got %q", right)
	}
}

func TestMarketsStatusBarShowsRefreshing(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.refreshing = true
	m.lastRefreshed = time.Now()

	right := m.statusRight()
	if right != "Refreshing" {
		t.Errorf("expected statusRight 'Refreshing', got %q", right)
	}
}

func TestMarketsStatusBarShowsError(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.lastErr = "some error"
	m.lastRefreshed = time.Now()

	right := m.statusRight()
	if right != "error: some error" {
		t.Errorf("expected statusRight 'error: some error', got %q", right)
	}
}

func TestMarketsStatusBarShowsSynced(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.lastRefreshed = time.Now()

	right := m.statusRight()
	if right != "Synced" {
		t.Errorf("expected statusRight 'Synced', got %q", right)
	}
}

func TestMarketsStatusBarShowsStale(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.lastRefreshed = time.Now().Add(-6 * time.Minute)

	right := m.statusRight()
	if right != "Stale" {
		t.Errorf("expected statusRight 'Stale', got %q", right)
	}
}

func TestMarketsTableRendersWhileError(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.lastRefreshed = time.Now()
	m.lastErr = "network failed"

	view := m.View()
	if !strings.Contains(view, "Coin 1") {
		t.Error("expected view to contain coin names even with error")
	}
	if !strings.Contains(view, "error: network failed") {
		t.Error("expected view to contain error text")
	}
}

func TestMarketsStatusBarHasHintsOnLeft(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.lastRefreshed = time.Now()

	view := m.View()
	if !strings.Contains(view, "j/k navigate") {
		t.Errorf("expected view to contain 'j/k navigate', got %q", view)
	}
}

func TestMarketsInitFetchesHundredCoinsOnFirstLaunch(t *testing.T) {
	coins := threeCoins()
	api := &StubAPI{coins: coins}
	s := &StubStore{}
	m := NewMarketsModel(testCtx, s, api)

	msg := executeInitBatchForMarkets(t, m)

	loaded, ok := msg.(coinsLoadedMsg)
	if !ok {
		t.Fatalf("expected coinsLoadedMsg, got %T: %v", msg, msg)
	}
	if len(loaded.coins) != 3 {
		t.Errorf("expected 3 coins, got %d", len(loaded.coins))
	}
	if len(api.fetchMarketsCalls) != 1 || api.fetchMarketsCalls[0] != 100 {
		t.Errorf("expected FetchMarkets called with 100, got %v", api.fetchMarketsCalls)
	}
}

func TestMarketsInitLoadsFromDBOnSubsequentLaunch(t *testing.T) {
	coins := makeCoins(100)
	api := &StubAPI{coins: coins}
	s := &StubStore{coins: coins}
	m := NewMarketsModel(testCtx, s, api)

	msg := executeInitBatchForMarkets(t, m)

	loaded, ok := msg.(coinsLoadedMsg)
	if !ok {
		t.Fatalf("expected coinsLoadedMsg, got %T: %v", msg, msg)
	}
	if len(loaded.coins) != 100 {
		t.Errorf("expected 100 coins from DB, got %d", len(loaded.coins))
	}
	if len(api.fetchMarketsCalls) != 0 {
		t.Errorf("expected no API calls, got %v", api.fetchMarketsCalls)
	}
}

func TestMarketsInitRefetchesWhenDBPartiallySeeded(t *testing.T) {
	partial := makeCoins(10)
	full := makeCoins(100)
	api := &StubAPI{coins: full}
	s := &StubStore{coins: partial}
	m := NewMarketsModel(testCtx, s, api)

	msg := executeInitBatchForMarkets(t, m)

	loaded, ok := msg.(coinsLoadedMsg)
	if !ok {
		t.Fatalf("expected coinsLoadedMsg, got %T: %v", msg, msg)
	}
	if len(loaded.coins) != 100 {
		t.Errorf("expected 100 coins after refetch, got %d", len(loaded.coins))
	}
	if len(api.fetchMarketsCalls) != 1 || api.fetchMarketsCalls[0] != 100 {
		t.Errorf("expected FetchMarkets called with 100, got %v", api.fetchMarketsCalls)
	}
}

func TestMarketsIgnoresOtherKeys(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	otherKeys := []rune{'a', 'b', 'c', 'x', 'z', '1', '2', ' '}
	for _, key := range otherKeys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}}
		_, cmd := m.update(msg)
		if cmd != nil {
			t.Errorf("expected nil cmd for key %q, got non-nil cmd", key)
		}
	}
}

func TestMarketsInputActiveFalse(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, api)
	if m.InputActive() {
		t.Error("expected InputActive() to return false for MarketsModel")
	}
}

func executeInitBatchForMarkets(t *testing.T, m MarketsModel) tea.Msg {
	t.Helper()
	cmd := m.Init()
	result := cmd()
	batch, ok := result.(tea.BatchMsg)
	if !ok {
		return result
	}
	for _, c := range batch {
		msg := c()
		switch msg.(type) {
		case coinsLoadedMsg, errMsg:
			return msg
		}
	}
	t.Fatal("no coinsLoadedMsg or errMsg in batch")
	return nil
}
