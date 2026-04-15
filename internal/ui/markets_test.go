package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredericomozzato/crypto_tracker/internal/api"
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

// Rate limiting tests

func TestMarketsRateLimitedErrMsg(t *testing.T) {
	stub := &StubStore{}
	apiStub := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, apiStub)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()

	rle := &api.RateLimitError{Body: "rate limit exceeded"}
	msg := errMsg{err: rle}
	updated, _ := m.update(msg)

	if updated.refreshAttempts != 1 {
		t.Errorf("expected refreshAttempts 1, got %d", updated.refreshAttempts)
	}

	if updated.rateLimitedUntil.IsZero() {
		t.Error("expected rateLimitedUntil to be set")
	}

	if time.Until(updated.rateLimitedUntil) <= 0 {
		t.Error("expected rateLimitedUntil to be in the future")
	}
}

func TestMarketsRateLimitedStatusBar(t *testing.T) {
	stub := &StubStore{}
	apiStub := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, apiStub)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.rateLimitedUntil = time.Now().Add(30 * time.Second)

	right := m.statusRight()
	if !strings.HasPrefix(right, "Rate limited") {
		t.Errorf("expected statusRight to start with 'Rate limited', got %q", right)
	}

	if !strings.Contains(right, "30s") && !strings.Contains(right, "29s") && !strings.Contains(right, "28s") {
		t.Errorf("expected statusRight to contain retry seconds, got %q", right)
	}
}

func TestMarketsRateLimitedStatusBarNoLongerLimited(t *testing.T) {
	stub := &StubStore{}
	apiStub := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, apiStub)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.lastRefreshed = time.Now()
	m.rateLimitedUntil = time.Now().Add(-30 * time.Second) // In the past

	right := m.statusRight()
	if right != "Synced" {
		t.Errorf("expected statusRight 'Synced', got %q", right)
	}
}

func TestMarketsRefreshBlockedWhenRateLimited(t *testing.T) {
	stub := &StubStore{}
	apiStub := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, apiStub)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.rateLimitedUntil = time.Now().Add(30 * time.Second)
	m.refreshing = false

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	updated, cmd := m.update(msg)

	if cmd != nil {
		t.Error("expected nil cmd when rate-limited")
	}

	if updated.refreshing {
		t.Error("expected refreshing to stay false")
	}
}

func TestMarketsAutoRefreshBlockedWhenRateLimited(t *testing.T) {
	stub := &StubStore{}
	apiStub := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, apiStub)
	m.coins = threeCoins()
	m.lastRefreshed = time.Now().Add(-61 * time.Second)
	m.refreshing = false
	m.rateLimitedUntil = time.Now().Add(30 * time.Second)

	updated, cmd := m.update(tickMsg(time.Now()))

	if updated.refreshing {
		t.Error("expected refreshing to stay false when rate-limited")
	}

	// cmdTick is always returned, but no refresh cmd should be in the batch
	if cmd == nil {
		t.Error("expected non-nil cmd from tickMsg (ticker should be re-armed)")
	}
}

func TestMarketsAutoRefreshResumesAfterCooldown(t *testing.T) {
	stub := &StubStore{coins: threeCoins()}
	apiStub := &StubAPI{prices: map[string]float64{"coin-1": 100.0}}
	m := NewMarketsModel(testCtx, stub, apiStub)
	m.coins = threeCoins()
	m.lastRefreshed = time.Now().Add(-61 * time.Second)
	m.refreshing = false
	m.rateLimitedUntil = time.Now().Add(-1 * time.Second) // Expired

	updated, cmd := m.update(tickMsg(time.Now()))

	if !updated.refreshing {
		t.Error("expected refreshing to be true when cooldown expired")
	}

	if cmd == nil {
		t.Error("expected non-nil cmd when refresh fires")
	}
}

func TestMarketsExponentialBackoffOnRepeated429(t *testing.T) {
	stub := &StubStore{}
	apiStub := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, apiStub)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()

	rle := &api.RateLimitError{Body: "rate limit exceeded"}
	var prevUntil time.Time

	for i := 0; i < 5; i++ {
		beforeUpdate := time.Now()
		updated, _ := m.update(errMsg{err: rle})

		cooldown := updated.rateLimitedUntil.Sub(beforeUpdate)

		switch i {
		case 0:
			if cooldown < 59*time.Second || cooldown > 61*time.Second {
				t.Errorf("attempt 1: expected cooldown ~60s, got %v", cooldown)
			}
		case 1:
			if cooldown < 119*time.Second || cooldown > 121*time.Second {
				t.Errorf("attempt 2: expected cooldown ~120s, got %v", cooldown)
			}
		case 2:
			if cooldown < 239*time.Second || cooldown > 241*time.Second {
				t.Errorf("attempt 3: expected cooldown ~240s, got %v", cooldown)
			}
		case 3:
			if cooldown < 299*time.Second || cooldown > 301*time.Second {
				t.Errorf("attempt 4: expected cooldown capped at ~300s, got %v", cooldown)
			}
		case 4:
			// Should still be capped at 300s
			if cooldown < 299*time.Second || cooldown > 301*time.Second {
				t.Errorf("attempt 5: expected cooldown capped at ~300s, got %v", cooldown)
			}
		}

		// Update m for next iteration, simulating time passing
		m = updated
		m.rateLimitedUntil = time.Now() // Reset so next error triggers new calculation

		if !prevUntil.IsZero() && m.rateLimitedUntil.Before(prevUntil) {
			// This is expected since we're resetting, just verifying the pattern
		}
		prevUntil = m.rateLimitedUntil
	}
}

func TestMarketsBackoffResetOnSuccess(t *testing.T) {
	stub := &StubStore{coins: threeCoins()}
	apiStub := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, apiStub)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.refreshing = true
	m.refreshAttempts = 3
	m.rateLimitedUntil = time.Now().Add(30 * time.Second)

	updated, _ := m.update(pricesUpdatedMsg{coins: threeCoins()})

	if updated.refreshAttempts != 0 {
		t.Errorf("expected refreshAttempts reset to 0, got %d", updated.refreshAttempts)
	}

	if !updated.rateLimitedUntil.IsZero() {
		t.Errorf("expected rateLimitedUntil to be zero time, got %v", updated.rateLimitedUntil)
	}
}

func TestMarketsNonRateLimitErrorDoesNotSetRateLimitedUntil(t *testing.T) {
	stub := &StubStore{}
	apiStub := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, apiStub)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()

	msg := errMsg{err: errors.New("network failed")}
	updated, _ := m.update(msg)

	if !updated.rateLimitedUntil.IsZero() {
		t.Errorf("expected rateLimitedUntil to remain zero, got %v", updated.rateLimitedUntil)
	}

	if updated.lastErr != "network failed" {
		t.Errorf("expected lastErr 'network failed', got %q", updated.lastErr)
	}
}

func TestMarketsRateLimitedStatusBarStyled(t *testing.T) {
	stub := &StubStore{}
	apiStub := &StubAPI{}
	m := NewMarketsModel(testCtx, stub, apiStub)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.rateLimitedUntil = time.Now().Add(30 * time.Second)

	view := m.View()
	if !strings.Contains(view, "Rate limited") {
		t.Errorf("expected view to contain 'Rate limited', got %q", view)
	}
}
