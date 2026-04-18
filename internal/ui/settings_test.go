package ui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

func TestNewSettingsModel(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewSettingsModel(context.Background(), stub, api)

	// Should start in browsing mode
	if m.InputActive() {
		t.Error("expected InputActive to be false in browsing mode")
	}

	// Should have default currency
	if m.selectedCode != "usd" {
		t.Errorf("expected default selectedCode 'usd', got %s", m.selectedCode)
	}
}

func TestSettingsInputActiveFalseWhenBrowsing(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewSettingsModel(context.Background(), stub, api)

	if m.InputActive() {
		t.Error("expected InputActive() to return false in browsing mode")
	}
}

func TestSettingsInputActiveTrueWhenPicking(t *testing.T) {
	stub := &StubStore{
		currencies: []store.Currency{
			{Code: "usd", Name: "US Dollar"},
			{Code: "eur", Name: "Euro"},
		},
	}
	api := &StubAPI{}
	m := NewSettingsModel(context.Background(), stub, api)

	// Simulate loading currencies and pressing Enter
	m.currencies = stub.currencies
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := m.update(msg)
	m = updated

	if !m.InputActive() {
		t.Error("expected InputActive() to return true in picking mode")
	}
}

func TestSettingsEnterOpensPickerWhenCurrenciesAvailable(t *testing.T) {
	stub := &StubStore{
		currencies: []store.Currency{
			{Code: "usd", Name: "US Dollar"},
			{Code: "eur", Name: "Euro"},
		},
	}
	api := &StubAPI{}
	m := NewSettingsModel(context.Background(), stub, api)
	m.currencies = stub.currencies

	// Should be in browsing mode initially
	if m.InputActive() {
		t.Error("expected browsing mode initially")
	}

	// Press Enter to open picker
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := m.update(msg)
	m = updated

	if !m.InputActive() {
		t.Error("expected picking mode after Enter")
	}
}

func TestSettingsEnterTriggersFetchWhenNoCurrencies(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{
		supportedCurrencies: []string{"usd", "eur", "btc"},
	}
	m := NewSettingsModel(context.Background(), stub, api)
	m.currencies = []store.Currency{} // Empty currencies

	// Press Enter with no currencies - should produce settingsNeedFetchMsg
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.update(msg)

	if cmd == nil {
		t.Fatal("expected command from Enter when currencies empty")
	}

	result := cmd()
	if _, ok := result.(settingsNeedFetchMsg); !ok {
		t.Errorf("expected settingsNeedFetchMsg, got %T", result)
	}
}

func TestSettingsPickArrowNavigation(t *testing.T) {
	currencies := []store.Currency{
		{Code: "usd", Name: "US Dollar"},
		{Code: "eur", Name: "Euro"},
		{Code: "gbp", Name: "British Pound"},
	}
	stub := &StubStore{currencies: currencies}
	api := &StubAPI{}
	m := NewSettingsModel(context.Background(), stub, api)
	m.currencies = currencies

	// Enter picking mode
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter})

	picking := m.mode.(settingsPicking)
	if picking.cursor != 0 {
		t.Errorf("expected cursor at 0 initially, got %d", picking.cursor)
	}

	// Press Down to move down
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyDown})
	picking = m.mode.(settingsPicking)
	if picking.cursor != 1 {
		t.Errorf("expected cursor at 1 after Down, got %d", picking.cursor)
	}

	// Press Up to move up
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyUp})
	picking = m.mode.(settingsPicking)
	if picking.cursor != 0 {
		t.Errorf("expected cursor at 0 after Up, got %d", picking.cursor)
	}
}

func TestSettingsPickFilterReducesList(t *testing.T) {
	currencies := []store.Currency{
		{Code: "usd", Name: "US Dollar"},
		{Code: "eur", Name: "Euro"},
		{Code: "gbp", Name: "British Pound"},
	}
	stub := &StubStore{currencies: currencies}
	api := &StubAPI{}
	m := NewSettingsModel(context.Background(), stub, api)
	m.currencies = currencies

	// Enter picking mode
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter})
	picking := m.mode.(settingsPicking)

	if len(picking.filtered) != 3 {
		t.Fatalf("expected 3 currencies initially, got %d", len(picking.filtered))
	}

	// Type "eu" to filter
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e', 'u'}})
	picking = m.mode.(settingsPicking)

	if len(picking.filtered) != 1 {
		t.Errorf("expected 1 currency after filtering 'eu', got %d", len(picking.filtered))
	}

	if picking.filtered[0].Code != "eur" {
		t.Errorf("expected eur, got %s", picking.filtered[0].Code)
	}
}

func TestSettingsPickEscReturnsToBrowsing(t *testing.T) {
	currencies := []store.Currency{
		{Code: "usd", Name: "US Dollar"},
	}
	stub := &StubStore{currencies: currencies}
	api := &StubAPI{}
	m := NewSettingsModel(context.Background(), stub, api)
	m.currencies = currencies

	// Enter picking mode
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.InputActive() {
		t.Fatal("expected picking mode")
	}

	// Press Esc
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEscape})

	if m.InputActive() {
		t.Error("expected browsing mode after Esc")
	}
}

func TestSettingsPickEnterSelectsCurrency(t *testing.T) {
	currencies := []store.Currency{
		{Code: "usd", Name: "US Dollar"},
		{Code: "eur", Name: "Euro"},
	}
	stub := &StubStore{currencies: currencies}
	api := &StubAPI{}
	m := NewSettingsModel(context.Background(), stub, api)
	m.currencies = currencies

	// Enter picking mode
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter})

	// Move cursor to second currency (EUR)
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyDown})

	// Press Enter to select
	m, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify command returns currencyChangedMsg
	if cmd == nil {
		t.Fatal("expected command from Enter in picking mode, got nil")
	}
	msg := cmd()
	changedMsg, ok := msg.(currencyChangedMsg)
	if !ok {
		t.Fatalf("expected currencyChangedMsg, got %T", msg)
	}
	if changedMsg.code != "eur" {
		t.Errorf("expected currency code 'eur', got '%s'", changedMsg.code)
	}

	// Verify mode transitioned to browsing
	if m.InputActive() {
		t.Error("expected browsing mode after selection, still in picking mode")
	}

	// Verify selectedCode updated
	if m.selectedCode != "eur" {
		t.Errorf("expected selectedCode 'eur', got '%s'", m.selectedCode)
	}

	// Verify SetSetting was called
	if stub.settings == nil || stub.settings["selected_currency"] != "eur" {
		t.Error("expected SetSetting to be called with selected_currency=eur")
	}
}

func TestSettingsPickCursorClampsAtTop(t *testing.T) {
	currencies := []store.Currency{
		{Code: "usd", Name: "US Dollar"},
		{Code: "eur", Name: "Euro"},
	}
	stub := &StubStore{currencies: currencies}
	api := &StubAPI{}
	m := NewSettingsModel(context.Background(), stub, api)
	m.currencies = currencies

	// Enter picking mode
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter})

	// Press Up when at top
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyUp})
	picking := m.mode.(settingsPicking)

	if picking.cursor != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", picking.cursor)
	}
}

func TestSettingsPickCursorClampsAtBottom(t *testing.T) {
	currencies := []store.Currency{
		{Code: "usd", Name: "US Dollar"},
		{Code: "eur", Name: "Euro"},
	}
	stub := &StubStore{currencies: currencies}
	api := &StubAPI{}
	m := NewSettingsModel(context.Background(), stub, api)
	m.currencies = currencies

	// Enter picking mode
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter})

	// Move to last item using Down arrow
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyDown})

	// Try to move past end
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyDown})
	picking := m.mode.(settingsPicking)

	if picking.cursor != 1 {
		t.Errorf("expected cursor to stay at 1, got %d", picking.cursor)
	}
}

func TestSettingsBrowsingShowsSelectedCurrency(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewSettingsModel(context.Background(), stub, api)
	m.selectedCode = "eur"
	m.currencies = []store.Currency{
		{Code: "eur", Name: "Euro"},
	}

	view := m.View()

	if !strings.Contains(view, "EUR") {
		t.Errorf("expected view to contain 'EUR', got %q", view)
	}
}

func TestSettingsLoadedMsgUpdatesState(t *testing.T) {
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewSettingsModel(context.Background(), stub, api)

	currencies := []store.Currency{
		{Code: "gbp", Name: "British Pound"},
	}

	msg := settingsLoadedMsg{
		currencies: currencies,
		selected:   "gbp",
	}

	updated, _ := m.update(msg)
	m = updated

	if len(m.currencies) != 1 {
		t.Errorf("expected 1 currency, got %d", len(m.currencies))
	}

	if m.selectedCode != "gbp" {
		t.Errorf("expected selectedCode 'gbp', got %s", m.selectedCode)
	}
}

func TestCurrenciesUpsertedMsgTransitionsToPicking(t *testing.T) {
	currencies := []store.Currency{
		{Code: "usd", Name: "US Dollar"},
	}
	stub := &StubStore{}
	api := &StubAPI{}
	m := NewSettingsModel(context.Background(), stub, api)
	m.loading = true

	msg := currenciesUpsertedMsg{currencies: currencies}
	updated, _ := m.update(msg)
	m = updated

	if m.loading {
		t.Error("expected loading to be false after currenciesUpsertedMsg")
	}

	if !m.InputActive() {
		t.Error("expected picking mode after currenciesUpsertedMsg")
	}

	if len(m.currencies) != 1 {
		t.Errorf("expected 1 currency, got %d", len(m.currencies))
	}
}

func TestFilterCurrenciesEmptyFilter(t *testing.T) {
	currencies := []store.Currency{
		{Code: "usd", Name: "US Dollar"},
		{Code: "eur", Name: "Euro"},
	}

	result := filterCurrencies(currencies, "")

	if len(result) != 2 {
		t.Errorf("expected 2 currencies with empty filter, got %d", len(result))
	}
}

func TestFilterCurrenciesByCode(t *testing.T) {
	currencies := []store.Currency{
		{Code: "usd", Name: "US Dollar"},
		{Code: "eur", Name: "Euro"},
		{Code: "gbp", Name: "British Pound"},
	}

	result := filterCurrencies(currencies, "us")

	if len(result) != 1 {
		t.Errorf("expected 1 currency matching 'us', got %d", len(result))
	}

	if result[0].Code != "usd" {
		t.Errorf("expected usd, got %s", result[0].Code)
	}
}

func TestFilterCurrenciesByName(t *testing.T) {
	currencies := []store.Currency{
		{Code: "usd", Name: "US Dollar"},
		{Code: "eur", Name: "Euro"},
		{Code: "gbp", Name: "British Pound"},
	}

	result := filterCurrencies(currencies, "pound")

	if len(result) != 1 {
		t.Errorf("expected 1 currency matching 'pound', got %d", len(result))
	}

	if result[0].Code != "gbp" {
		t.Errorf("expected gbp, got %s", result[0].Code)
	}
}

func TestFilterCurrenciesNoMatches(t *testing.T) {
	currencies := []store.Currency{
		{Code: "usd", Name: "US Dollar"},
		{Code: "eur", Name: "Euro"},
	}

	result := filterCurrencies(currencies, "xyz")

	if len(result) != 0 {
		t.Errorf("expected 0 currencies matching 'xyz', got %d", len(result))
	}
}
