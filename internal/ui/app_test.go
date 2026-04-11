package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

// StubStore implements store.Store for testing
type StubStore struct {
	coins []store.Coin
	err   error
}

func (s *StubStore) UpsertCoin(ctx context.Context, c store.Coin) error {
	if s.err != nil {
		return s.err
	}
	for i, existing := range s.coins {
		if existing.ApiID == c.ApiID {
			s.coins[i] = c
			return nil
		}
	}
	s.coins = append(s.coins, c)
	return nil
}

func (s *StubStore) GetAllCoins(ctx context.Context) ([]store.Coin, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.coins, nil
}

func (s *StubStore) Close() error {
	return nil
}

func (s *StubStore) UpdatePrices(ctx context.Context, prices map[string]float64) error {
	return s.err
}

// StubAPI implements api.CoinGeckoClient for testing
type StubAPI struct {
	coins             []store.Coin
	prices            map[string]float64
	err               error
	fetchMarketsCalls []int // records the limit arg each time FetchMarkets is called
}

func (a *StubAPI) FetchMarkets(ctx context.Context, limit int) ([]store.Coin, error) {
	a.fetchMarketsCalls = append(a.fetchMarketsCalls, limit)
	if a.err != nil {
		return nil, a.err
	}
	return a.coins, nil
}

func (a *StubAPI) FetchPrices(ctx context.Context, apiIDs []string) (map[string]float64, error) {
	if a.err != nil {
		return nil, a.err
	}
	return a.prices, nil
}

func TestNewAppModel(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	// Verify the model renders without panicking; a zero-dimension model
	// should still produce output (the "too small" message).
	view := m.View()
	if view == "" {
		t.Error("expected non-empty View from NewAppModel, got empty string")
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

func TestWindowSizeMsg(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)

	model, ok := updated.(AppModel)
	if !ok {
		t.Fatalf("expected AppModel, got %T", updated)
	}

	if model.width != 120 {
		t.Errorf("expected width 120, got %d", model.width)
	}
	if model.height != 40 {
		t.Errorf("expected height 40, got %d", model.height)
	}
}

func TestIgnoresOtherKeys(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	otherKeys := []rune{'a', 'b', 'c', 'x', 'z', '1', ' '}
	for _, key := range otherKeys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}}
		_, cmd := m.Update(msg)
		if cmd != nil {
			t.Errorf("expected nil cmd for key %q, got non-nil cmd", key)
		}
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

func TestCoinsLoadedMsg(t *testing.T) {
	s := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), s, api)
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
	updated, _ := m.Update(msg)

	model, ok := updated.(AppModel)
	if !ok {
		t.Fatalf("expected AppModel, got %T", updated)
	}

	view := model.View()
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

func TestErrMsg(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30

	testErr := "connection failed"
	msg := errMsg{err: errors.New(testErr)}
	updated, _ := m.Update(msg)

	model, ok := updated.(AppModel)
	if !ok {
		t.Fatalf("expected AppModel, got %T", updated)
	}

	view := model.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}

	if !strings.Contains(view, testErr) {
		t.Errorf("expected view to contain error %q, got %q", testErr, view)
	}
}

func TestViewRendersLoading(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
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

func TestRefreshKeyReturnsCmdWhenCoinsLoaded(t *testing.T) {
	storeStub := &StubStore{coins: []store.Coin{{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC", Rate: 67000.00}}}
	api := &StubAPI{prices: map[string]float64{"bitcoin": 68000.00}}
	m := NewAppModel(context.Background(), storeStub, api)
	m.width = 100
	m.height = 30

	// First load coins
	updated, _ := m.Update(coinsLoadedMsg{coins: storeStub.coins})
	m = updated.(AppModel)

	// Then press 'r'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	updated, cmd := m.Update(msg)

	model, ok := updated.(AppModel)
	if !ok {
		t.Fatalf("expected AppModel, got %T", updated)
	}

	if !model.refreshing {
		t.Error("expected refreshing to be true")
	}

	if cmd == nil {
		t.Fatal("expected non-nil cmd when pressing r with coins loaded")
	}
}

func TestRefreshKeyIgnoredWhenAlreadyRefreshing(t *testing.T) {
	stub := &StubStore{coins: []store.Coin{{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC", Rate: 67000.00}}}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30
	m.coins = stub.coins
	m.refreshing = true

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	_, cmd := m.Update(msg)

	if cmd != nil {
		t.Error("expected nil cmd when already refreshing")
	}
}

func TestRefreshKeyIgnoredWhenNoCoins(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	_, cmd := m.Update(msg)

	if cmd != nil {
		t.Error("expected nil cmd when no coins loaded")
	}
}

func TestPricesUpdatedMsg(t *testing.T) {
	storeStub := &StubStore{coins: []store.Coin{{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC", Rate: 68000.00}}}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), storeStub, api)
	m.width = 100
	m.height = 30
	m.coins = []store.Coin{{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC", Rate: 67000.00}}
	m.refreshing = true

	msg := pricesUpdatedMsg{coins: storeStub.coins}
	updated, _ := m.Update(msg)

	model, ok := updated.(AppModel)
	if !ok {
		t.Fatalf("expected AppModel, got %T", updated)
	}

	if model.refreshing {
		t.Error("expected refreshing to be false after pricesUpdatedMsg")
	}

	if len(model.coins) != 1 || model.coins[0].Rate != 68000.00 {
		t.Errorf("expected updated coin with Rate 68000.00, got %v", model.coins)
	}
}

func TestViewShowsRefreshHint(t *testing.T) {
	stub := &StubStore{coins: []store.Coin{{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC", Rate: 67000.00, MarketRank: 1}}}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30
	m.coins = stub.coins

	view := m.View()
	if !strings.Contains(view, "r refresh") {
		t.Errorf("expected view to contain 'r refresh', got %q", view)
	}
}

func TestViewRendersColumnHeaders(t *testing.T) {
	coins := threeCoins()
	m := NewAppModel(context.Background(), &StubStore{coins: coins}, &StubAPI{})
	m.width = 120
	m.height = 40
	updated, _ := m.Update(coinsLoadedMsg{coins: coins})
	model := updated.(AppModel)

	view := model.View()
	for _, col := range []string{"#", "Name", "Ticker", "Price (USD)", "24h"} {
		if !strings.Contains(view, col) {
			t.Errorf("expected view to contain header %q, got %q", col, view)
		}
	}
}

func TestViewRendersHintLine(t *testing.T) {
	coins := threeCoins()
	m := NewAppModel(context.Background(), &StubStore{coins: coins}, &StubAPI{})
	m.width = 120
	m.height = 40
	updated, _ := m.Update(coinsLoadedMsg{coins: coins})
	model := updated.(AppModel)

	view := model.View()
	if !strings.Contains(view, "j/k") {
		t.Errorf("expected view to contain 'j/k', got %q", view)
	}
}

func TestInitFetchesHundredCoinsOnFirstLaunch(t *testing.T) {
	coins := threeCoins()
	api := &StubAPI{coins: coins}
	s := &StubStore{}
	m := NewAppModel(context.Background(), s, api)

	msg := executeInitBatch(t, m)

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

func TestInitLoadsFromDBOnSubsequentLaunch(t *testing.T) {
	coins := makeCoins(100)
	api := &StubAPI{coins: coins}
	s := &StubStore{coins: coins}
	m := NewAppModel(context.Background(), s, api)

	msg := executeInitBatch(t, m)

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

func TestInitRefetchesWhenDBPartiallySeeded(t *testing.T) {
	partial := makeCoins(10)
	full := makeCoins(100)
	api := &StubAPI{coins: full}
	s := &StubStore{coins: partial}
	m := NewAppModel(context.Background(), s, api)

	msg := executeInitBatch(t, m)

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

// executeInitBatch runs the batched command from Init() and returns the first
// coinsLoadedMsg or errMsg found.
func executeInitBatch(t *testing.T, m AppModel) tea.Msg {
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

func threeCoins() []store.Coin {
	return makeCoins(3)
}

func makeCoins(n int) []store.Coin {
	coins := make([]store.Coin, n)
	for i := range coins {
		coins[i] = store.Coin{
			ApiID:      fmt.Sprintf("coin-%d", i+1),
			Name:       fmt.Sprintf("Coin %d", i+1),
			Ticker:     fmt.Sprintf("C%d", i+1),
			Rate:       float64((i + 1) * 100),
			MarketRank: i + 1,
		}
	}
	return coins
}

func setupCursorModel(t *testing.T, coins []store.Coin) AppModel {
	t.Helper()
	m := NewAppModel(context.Background(), &StubStore{coins: coins}, &StubAPI{})
	m.width = 120
	m.height = 40
	updated, _ := m.Update(coinsLoadedMsg{coins: coins})
	return updated.(AppModel)
}

func TestCursorMovesDownOnJ(t *testing.T) {
	m := setupCursorModel(t, threeCoins())
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(AppModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 after 'j', got %d", m.cursor)
	}
}

func TestCursorMovesUpOnK(t *testing.T) {
	m := setupCursorModel(t, threeCoins())
	m.cursor = 1
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(AppModel)
	if m.cursor != 0 {
		t.Errorf("expected cursor 0 after 'k', got %d", m.cursor)
	}
}

func TestCursorClampsAtBottom(t *testing.T) {
	m := setupCursorModel(t, threeCoins())
	m.cursor = 2
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(AppModel)
	if m.cursor != 2 {
		t.Errorf("expected cursor 2 (clamped), got %d", m.cursor)
	}
}

func TestCursorClampsAtTop(t *testing.T) {
	m := setupCursorModel(t, threeCoins())
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(AppModel)
	if m.cursor != 0 {
		t.Errorf("expected cursor 0 (clamped), got %d", m.cursor)
	}
}

func TestCursorJumpsToTopOnG(t *testing.T) {
	m := setupCursorModel(t, threeCoins())
	m.cursor = 2
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(AppModel)
	if m.cursor != 0 {
		t.Errorf("expected cursor 0 after 'g', got %d", m.cursor)
	}
}

func TestCursorJumpsToBottomOnCapG(t *testing.T) {
	m := setupCursorModel(t, threeCoins())
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(AppModel)
	if m.cursor != 2 {
		t.Errorf("expected cursor 2 after 'G', got %d", m.cursor)
	}
}

func TestCursorMovesDownOnDownArrow(t *testing.T) {
	m := setupCursorModel(t, threeCoins())
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(AppModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 after KeyDown, got %d", m.cursor)
	}
}

func TestCursorMovesUpOnUpArrow(t *testing.T) {
	m := setupCursorModel(t, threeCoins())
	m.cursor = 1
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(AppModel)
	if m.cursor != 0 {
		t.Errorf("expected cursor 0 after KeyUp, got %d", m.cursor)
	}
}

func TestMoveCursorNoPanicOnEmptyCoins(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model := updated.(AppModel)
	if model.cursor != 0 {
		t.Errorf("expected cursor 0 on empty coins, got %d", model.cursor)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = updated.(AppModel)
	if model.cursor != 0 {
		t.Errorf("expected cursor 0 on empty coins after 'k', got %d", model.cursor)
	}
}

func TestCursorClampedAfterCoinsLoaded(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30

	m.cursor = 5
	updated, _ := m.Update(coinsLoadedMsg{coins: threeCoins()})
	model := updated.(AppModel)
	if model.cursor != 2 {
		t.Errorf("expected cursor clamped to 2 (last index), got %d", model.cursor)
	}
}

func TestTickMsgAlwaysReissuesTicker(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)

	updated, cmd := m.Update(tickMsg(time.Now()))
	model := updated.(AppModel)

	if cmd == nil {
		t.Error("expected non-nil cmd from tickMsg (ticker should be re-armed)")
	}
	_ = model
}

func TestTickMsgBelow60sDoesNotRefresh(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.coins = threeCoins()
	m.lastRefreshed = time.Now().Add(-30 * time.Second)

	updated, _ := m.Update(tickMsg(time.Now()))
	model := updated.(AppModel)

	if model.refreshing {
		t.Error("expected refreshing to stay false when 60s not elapsed")
	}
}

func TestTickMsgAbove60sFiresRefresh(t *testing.T) {
	stub := &StubStore{coins: threeCoins()}
	api := &StubAPI{prices: map[string]float64{"coin-1": 100.0}}
	m := NewAppModel(context.Background(), stub, api)
	m.coins = threeCoins()
	m.lastRefreshed = time.Now().Add(-61 * time.Second)
	m.refreshing = false

	updated, cmd := m.Update(tickMsg(time.Now()))
	model := updated.(AppModel)

	if !model.refreshing {
		t.Error("expected refreshing to be true when 60+ seconds elapsed")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd when refresh fires")
	}
}

func TestTickMsgWhenAlreadyRefreshing(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.coins = threeCoins()
	m.lastRefreshed = time.Now().Add(-61 * time.Second)
	m.refreshing = true

	updated, cmd := m.Update(tickMsg(time.Now()))
	model := updated.(AppModel)

	if !model.refreshing {
		t.Error("expected refreshing to remain true")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (ticker re-arm) even when already refreshing")
	}
}

func TestTickMsgWhenNoCoins(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.lastRefreshed = time.Now().Add(-61 * time.Second)

	updated, _ := m.Update(tickMsg(time.Now()))
	model := updated.(AppModel)

	if model.refreshing {
		t.Error("expected no refresh when no coins loaded")
	}
}

func TestCoinsLoadedSetsLastRefreshed(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30

	if !m.lastRefreshed.IsZero() {
		t.Error("expected lastRefreshed to be zero initially")
	}

	updated, _ := m.Update(coinsLoadedMsg{coins: threeCoins()})
	model := updated.(AppModel)

	if model.lastRefreshed.IsZero() {
		t.Error("expected lastRefreshed to be set after coinsLoadedMsg")
	}
}

func TestPricesUpdatedSetsLastRefreshed(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.refreshing = true

	updated, _ := m.Update(pricesUpdatedMsg{coins: threeCoins()})
	model := updated.(AppModel)

	if model.lastRefreshed.IsZero() {
		t.Error("expected lastRefreshed to be set after pricesUpdatedMsg")
	}
	if model.refreshing {
		t.Error("expected refreshing to be false after pricesUpdatedMsg")
	}
}

func TestStatusBarShowsLoading(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30

	right := m.statusRight()
	if right != "loading..." {
		t.Errorf("expected statusRight 'loading...', got %q", right)
	}
}

func TestStatusBarShowsRefreshing(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
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

func TestStatusBarShowsError(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
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

func TestStatusBarShowsSynced(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.lastRefreshed = time.Now()

	right := m.statusRight()
	if right != "Synced" {
		t.Errorf("expected statusRight 'Synced', got %q", right)
	}
}

func TestStatusBarShowsStale(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.lastRefreshed = time.Now().Add(-6 * time.Minute)

	right := m.statusRight()
	if right != "Stale" {
		t.Errorf("expected statusRight 'Stale', got %q", right)
	}
}

func TestTableRendersWhileError(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
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

func TestStatusBarHasHintsOnLeft(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30
	m.coins = threeCoins()
	m.lastRefreshed = time.Now()

	view := m.View()
	if !strings.Contains(view, "j/k navigate") {
		t.Errorf("expected view to contain 'j/k navigate', got %q", view)
	}
}
