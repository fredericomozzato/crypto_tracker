package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

// StubStore implements store.Store for testing
type StubStore struct {
	coins []store.Coin
	err   error
}

func (s *StubStore) UpsertCoin(ctx context.Context, c store.Coin) error {
	return s.err
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
	coins  []store.Coin
	prices map[string]float64
	err    error
}

func (a *StubAPI) FetchMarkets(ctx context.Context, limit int) ([]store.Coin, error) {
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

func TestInitReturnsCmd(t *testing.T) {
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

	if !strings.Contains(view, "67000") {
		t.Errorf("expected view to contain price, got %q", view)
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
	stub := &StubStore{coins: []store.Coin{{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC", Rate: 67000.00}}}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30
	m.coins = stub.coins

	view := m.View()
	if !strings.Contains(view, "r to refresh") {
		t.Errorf("expected view to contain 'r to refresh', got %q", view)
	}
}

func TestViewShowsRefreshingIndicator(t *testing.T) {
	stub := &StubStore{coins: []store.Coin{{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC", Rate: 67000.00}}}
	api := &StubAPI{}
	m := NewAppModel(context.Background(), stub, api)
	m.width = 100
	m.height = 30
	m.coins = stub.coins
	m.refreshing = true

	view := m.View()
	if !strings.Contains(view, "refreshing...") {
		t.Errorf("expected view to contain 'refreshing...', got %q", view)
	}
}
