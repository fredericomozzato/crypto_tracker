package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fredericomozzato/crypto_tracker/internal/db"
)

func TestUpsertAndReadBack(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	coin := Coin{
		ApiID:       "bitcoin",
		Name:        "Bitcoin",
		Ticker:      "BTC",
		Rate:        67000.00,
		PriceChange: -1.23,
		MarketRank:  1,
	}

	if err := s.UpsertCoin(ctx, coin); err != nil {
		t.Fatalf("failed to upsert coin: %v", err)
	}

	coins, err := s.GetAllCoins(ctx)
	if err != nil {
		t.Fatalf("failed to get all coins: %v", err)
	}

	if len(coins) != 1 {
		t.Fatalf("expected 1 coin, got %d", len(coins))
	}

	c := coins[0]
	if c.ApiID != "bitcoin" {
		t.Errorf("expected ApiID 'bitcoin', got %s", c.ApiID)
	}
	if c.Name != "Bitcoin" {
		t.Errorf("expected Name 'Bitcoin', got %s", c.Name)
	}
	if c.Ticker != "BTC" {
		t.Errorf("expected Ticker 'BTC', got %s", c.Ticker)
	}
	if c.Rate != 67000.00 {
		t.Errorf("expected Rate 67000.00, got %f", c.Rate)
	}
	if c.PriceChange != -1.23 {
		t.Errorf("expected PriceChange -1.23, got %f", c.PriceChange)
	}
	if c.MarketRank != 1 {
		t.Errorf("expected MarketRank 1, got %d", c.MarketRank)
	}
	if c.UpdatedAt == 0 {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestUpsertUpdatesExisting(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	coin1 := Coin{
		ApiID:       "bitcoin",
		Name:        "Bitcoin",
		Ticker:      "BTC",
		Rate:        67000.00,
		PriceChange: -1.23,
		MarketRank:  1,
	}

	if err := s.UpsertCoin(ctx, coin1); err != nil {
		t.Fatalf("failed to upsert first coin: %v", err)
	}

	coin2 := Coin{
		ApiID:       "bitcoin",
		Name:        "Bitcoin",
		Ticker:      "BTC",
		Rate:        68000.00,
		PriceChange: 2.5,
		MarketRank:  1,
	}

	if err := s.UpsertCoin(ctx, coin2); err != nil {
		t.Fatalf("failed to upsert second coin: %v", err)
	}

	coins, err := s.GetAllCoins(ctx)
	if err != nil {
		t.Fatalf("failed to get all coins: %v", err)
	}

	if len(coins) != 1 {
		t.Fatalf("expected 1 coin, got %d", len(coins))
	}

	c := coins[0]
	if c.Rate != 68000.00 {
		t.Errorf("expected Rate 68000.00, got %f", c.Rate)
	}
	if c.PriceChange != 2.5 {
		t.Errorf("expected PriceChange 2.5, got %f", c.PriceChange)
	}
}

func TestGetAllCoinsOrdering(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	coins := []Coin{
		{ApiID: "ethereum", Name: "Ethereum", Ticker: "ETH", Rate: 3500, MarketRank: 2},
		{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC", Rate: 67000, MarketRank: 1},
		{ApiID: "tether", Name: "Tether", Ticker: "USDT", Rate: 1, MarketRank: 3},
	}

	for _, c := range coins {
		if err := s.UpsertCoin(ctx, c); err != nil {
			t.Fatalf("failed to upsert coin %s: %v", c.ApiID, err)
		}
	}

	result, err := s.GetAllCoins(ctx)
	if err != nil {
		t.Fatalf("failed to get all coins: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 coins, got %d", len(result))
	}

	expectedOrder := []string{"bitcoin", "ethereum", "tether"}
	for i, c := range result {
		if c.ApiID != expectedOrder[i] {
			t.Errorf("position %d: expected %s, got %s", i, expectedOrder[i], c.ApiID)
		}
	}
}

func TestGetAllCoinsEmpty(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	coins, err := s.GetAllCoins(ctx)
	if err != nil {
		t.Fatalf("failed to get all coins: %v", err)
	}

	if coins == nil {
		t.Error("expected non-nil slice, got nil")
	}

	if len(coins) != 0 {
		t.Errorf("expected 0 coins, got %d", len(coins))
	}
}

func TestUpdatePrices(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Upsert two coins
	coins := []Coin{
		{ApiID: "bitcoin", Name: "Bitcoin", Ticker: "BTC", Rate: 67000.00, MarketRank: 1},
		{ApiID: "ethereum", Name: "Ethereum", Ticker: "ETH", Rate: 3500.00, MarketRank: 2},
	}

	for _, c := range coins {
		if err := s.UpsertCoin(ctx, c); err != nil {
			t.Fatalf("failed to upsert coin %s: %v", c.ApiID, err)
		}
	}

	// Update prices
	newPrices := map[string]float64{
		"bitcoin":  68000.00,
		"ethereum": 3600.00,
	}

	if err := s.UpdatePrices(ctx, newPrices); err != nil {
		t.Fatalf("failed to update prices: %v", err)
	}

	// Read back and verify
	updatedCoins, err := s.GetAllCoins(ctx)
	if err != nil {
		t.Fatalf("failed to get all coins: %v", err)
	}

	for _, c := range updatedCoins {
		switch c.ApiID {
		case "bitcoin":
			if c.Rate != 68000.00 {
				t.Errorf("expected bitcoin Rate 68000.00, got %f", c.Rate)
			}
		case "ethereum":
			if c.Rate != 3600.00 {
				t.Errorf("expected ethereum Rate 3600.00, got %f", c.Rate)
			}
		}

		if c.UpdatedAt == 0 {
			t.Errorf("expected UpdatedAt to be set for %s", c.ApiID)
		}
	}
}

func TestUpdatePricesUnknownCoin(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Update prices for a coin that doesn't exist - should not error
	prices := map[string]float64{
		"unknown-coin": 100.00,
	}

	if err := s.UpdatePrices(ctx, prices); err != nil {
		t.Fatalf("expected no error for unknown coin, got: %v", err)
	}
}

func TestUpdatePricesEmpty(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Empty prices map - should not error
	if err := s.UpdatePrices(ctx, map[string]float64{}); err != nil {
		t.Fatalf("expected no error for empty prices, got: %v", err)
	}
}

func TestCreatePortfolio(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	_, err = s.CreatePortfolio(ctx, "Long Term")
	if err != nil {
		t.Fatalf("failed to create portfolio: %v", err)
	}

	portfolios, err := s.GetAllPortfolios(ctx)
	if err != nil {
		t.Fatalf("failed to get all portfolios: %v", err)
	}

	if len(portfolios) != 1 {
		t.Fatalf("expected 1 portfolio, got %d", len(portfolios))
	}

	if portfolios[0].Name != "Long Term" {
		t.Errorf("expected Name 'Long Term', got %s", portfolios[0].Name)
	}

	if portfolios[0].ID <= 0 {
		t.Errorf("expected ID > 0, got %d", portfolios[0].ID)
	}
}

func TestCreatePortfolioSetsCreatedAt(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	p, err := s.CreatePortfolio(ctx, "Test Portfolio")
	if err != nil {
		t.Fatalf("failed to create portfolio: %v", err)
	}

	if p.CreatedAt == 0 {
		t.Error("expected CreatedAt to be set")
	}

	portfolios, err := s.GetAllPortfolios(ctx)
	if err != nil {
		t.Fatalf("failed to get all portfolios: %v", err)
	}

	if portfolios[0].CreatedAt == 0 {
		t.Error("expected CreatedAt to be set when read back")
	}
}

func TestGetAllPortfoliosEmpty(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	portfolios, err := s.GetAllPortfolios(ctx)
	if err != nil {
		t.Fatalf("failed to get all portfolios: %v", err)
	}

	if portfolios == nil {
		t.Error("expected non-nil slice, got nil")
	}

	if len(portfolios) != 0 {
		t.Errorf("expected 0 portfolios, got %d", len(portfolios))
	}
}

func TestGetAllPortfoliosMultiple(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	names := []string{"A", "B", "C"}
	for _, name := range names {
		if _, err := s.CreatePortfolio(ctx, name); err != nil {
			t.Fatalf("failed to create portfolio %s: %v", name, err)
		}
	}

	portfolios, err := s.GetAllPortfolios(ctx)
	if err != nil {
		t.Fatalf("failed to get all portfolios: %v", err)
	}

	if len(portfolios) != 3 {
		t.Fatalf("expected 3 portfolios, got %d", len(portfolios))
	}

	for i, name := range names {
		if portfolios[i].Name != name {
			t.Errorf("position %d: expected %s, got %s", i, name, portfolios[i].Name)
		}
	}
}

func TestCreatePortfolioReturnsInsertedID(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	p1, err := s.CreatePortfolio(ctx, "First")
	if err != nil {
		t.Fatalf("failed to create first portfolio: %v", err)
	}

	p2, err := s.CreatePortfolio(ctx, "Second")
	if err != nil {
		t.Fatalf("failed to create second portfolio: %v", err)
	}

	if p2.ID <= p1.ID {
		t.Errorf("expected second ID (%d) > first ID (%d)", p2.ID, p1.ID)
	}
}

func TestUpsertHoldingInsertsNew(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Create a coin and portfolio first
	coin := Coin{
		ApiID:  "bitcoin",
		Name:   "Bitcoin",
		Ticker: "BTC",
		Rate:   67000.00,
	}
	if err := s.UpsertCoin(ctx, coin); err != nil {
		t.Fatalf("failed to upsert coin: %v", err)
	}

	coins, err := s.GetAllCoins(ctx)
	if err != nil {
		t.Fatalf("failed to get coins: %v", err)
	}
	coinID := coins[0].ID

	p, err := s.CreatePortfolio(ctx, "Test Portfolio")
	if err != nil {
		t.Fatalf("failed to create portfolio: %v", err)
	}

	// Insert holding
	if err := s.UpsertHolding(ctx, p.ID, coinID, 1.5); err != nil {
		t.Fatalf("failed to upsert holding: %v", err)
	}

	// Read back
	holdings, err := s.GetHoldingsForPortfolio(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get holdings: %v", err)
	}

	if len(holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(holdings))
	}

	h := holdings[0]
	if h.Amount != 1.5 {
		t.Errorf("expected Amount 1.5, got %f", h.Amount)
	}
	if h.CoinID != coinID {
		t.Errorf("expected CoinID %d, got %d", coinID, h.CoinID)
	}
	if h.Name != "Bitcoin" {
		t.Errorf("expected Name 'Bitcoin', got %s", h.Name)
	}
	if h.Ticker != "BTC" {
		t.Errorf("expected Ticker 'BTC', got %s", h.Ticker)
	}
}

func TestUpsertHoldingUpdatesOnConflict(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Create a coin and portfolio
	coin := Coin{
		ApiID:  "bitcoin",
		Name:   "Bitcoin",
		Ticker: "BTC",
		Rate:   67000.00,
	}
	if err := s.UpsertCoin(ctx, coin); err != nil {
		t.Fatalf("failed to upsert coin: %v", err)
	}

	coins, err := s.GetAllCoins(ctx)
	if err != nil {
		t.Fatalf("failed to get coins: %v", err)
	}
	coinID := coins[0].ID

	p, err := s.CreatePortfolio(ctx, "Test Portfolio")
	if err != nil {
		t.Fatalf("failed to create portfolio: %v", err)
	}

	// Insert holding
	if err := s.UpsertHolding(ctx, p.ID, coinID, 1.0); err != nil {
		t.Fatalf("failed to upsert holding: %v", err)
	}

	// Upsert same holding with different amount
	if err := s.UpsertHolding(ctx, p.ID, coinID, 2.5); err != nil {
		t.Fatalf("failed to upsert holding again: %v", err)
	}

	// Read back
	holdings, err := s.GetHoldingsForPortfolio(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get holdings: %v", err)
	}

	if len(holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(holdings))
	}

	if holdings[0].Amount != 2.5 {
		t.Errorf("expected Amount 2.5, got %f", holdings[0].Amount)
	}
}

func TestGetHoldingsForPortfolioEmpty(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	p, err := s.CreatePortfolio(ctx, "Test Portfolio")
	if err != nil {
		t.Fatalf("failed to create portfolio: %v", err)
	}

	holdings, err := s.GetHoldingsForPortfolio(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get holdings: %v", err)
	}

	if holdings == nil {
		t.Error("expected non-nil slice, got nil")
	}

	if len(holdings) != 0 {
		t.Errorf("expected 0 holdings, got %d", len(holdings))
	}
}

func TestGetHoldingsForPortfolioJoinsCoinData(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Create coin with all fields
	coin := Coin{
		ApiID:       "ethereum",
		Name:        "Ethereum",
		Ticker:      "ETH",
		Rate:        3500.00,
		PriceChange: 2.5,
	}
	if err := s.UpsertCoin(ctx, coin); err != nil {
		t.Fatalf("failed to upsert coin: %v", err)
	}

	coins, err := s.GetAllCoins(ctx)
	if err != nil {
		t.Fatalf("failed to get coins: %v", err)
	}
	coinID := coins[0].ID

	p, err := s.CreatePortfolio(ctx, "Test Portfolio")
	if err != nil {
		t.Fatalf("failed to create portfolio: %v", err)
	}

	if err := s.UpsertHolding(ctx, p.ID, coinID, 5.0); err != nil {
		t.Fatalf("failed to upsert holding: %v", err)
	}

	holdings, err := s.GetHoldingsForPortfolio(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get holdings: %v", err)
	}

	if len(holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(holdings))
	}

	h := holdings[0]
	if h.Name != "Ethereum" {
		t.Errorf("expected Name 'Ethereum', got %s", h.Name)
	}
	if h.Ticker != "ETH" {
		t.Errorf("expected Ticker 'ETH', got %s", h.Ticker)
	}
	if h.Rate != 3500.00 {
		t.Errorf("expected Rate 3500.00, got %f", h.Rate)
	}
	if h.PriceChange != 2.5 {
		t.Errorf("expected PriceChange 2.5, got %f", h.PriceChange)
	}
}

func TestGetHoldingsComputedValue(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Create coin with rate=500
	coin := Coin{
		ApiID:  "testcoin",
		Name:   "Test Coin",
		Ticker: "TEST",
		Rate:   500.00,
	}
	if err := s.UpsertCoin(ctx, coin); err != nil {
		t.Fatalf("failed to upsert coin: %v", err)
	}

	coins, err := s.GetAllCoins(ctx)
	if err != nil {
		t.Fatalf("failed to get coins: %v", err)
	}
	coinID := coins[0].ID

	p, err := s.CreatePortfolio(ctx, "Test Portfolio")
	if err != nil {
		t.Fatalf("failed to create portfolio: %v", err)
	}

	// amount=2, rate=500 → value=1000
	if err := s.UpsertHolding(ctx, p.ID, coinID, 2.0); err != nil {
		t.Fatalf("failed to upsert holding: %v", err)
	}

	holdings, err := s.GetHoldingsForPortfolio(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get holdings: %v", err)
	}

	if len(holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(holdings))
	}

	if holdings[0].Value != 1000.0 {
		t.Errorf("expected Value 1000.0, got %f", holdings[0].Value)
	}
}

func TestGetHoldingsOrderedByValueDesc(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Create coins with different rates
	coins := []Coin{
		{ApiID: "coin-a", Name: "Coin A", Ticker: "A", Rate: 100.00}, // value=100
		{ApiID: "coin-b", Name: "Coin B", Ticker: "B", Rate: 300.00}, // value=300
		{ApiID: "coin-c", Name: "Coin C", Ticker: "C", Rate: 200.00}, // value=200
	}
	for _, c := range coins {
		if err := s.UpsertCoin(ctx, c); err != nil {
			t.Fatalf("failed to upsert coin: %v", err)
		}
	}

	storedCoins, err := s.GetAllCoins(ctx)
	if err != nil {
		t.Fatalf("failed to get coins: %v", err)
	}

	p, err := s.CreatePortfolio(ctx, "Test Portfolio")
	if err != nil {
		t.Fatalf("failed to create portfolio: %v", err)
	}

	// Add holdings with amount=1 each
	for _, c := range storedCoins {
		if err := s.UpsertHolding(ctx, p.ID, c.ID, 1.0); err != nil {
			t.Fatalf("failed to upsert holding: %v", err)
		}
	}

	holdings, err := s.GetHoldingsForPortfolio(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get holdings: %v", err)
	}

	if len(holdings) != 3 {
		t.Fatalf("expected 3 holdings, got %d", len(holdings))
	}

	// Should be ordered by value desc: B (300), C (200), A (100)
	expectedOrder := []string{"Coin B", "Coin C", "Coin A"}
	for i, h := range holdings {
		if h.Name != expectedOrder[i] {
			t.Errorf("position %d: expected %s, got %s", i, expectedOrder[i], h.Name)
		}
	}
}

func TestGetHoldingsProportion(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Create coins: A with value=1000, B with value=3000 (total=4000)
	coins := []Coin{
		{ApiID: "coin-a", Name: "Coin A", Ticker: "A", Rate: 1000.00},
		{ApiID: "coin-b", Name: "Coin B", Ticker: "B", Rate: 3000.00},
	}
	for _, c := range coins {
		if err := s.UpsertCoin(ctx, c); err != nil {
			t.Fatalf("failed to upsert coin: %v", err)
		}
	}

	storedCoins, err := s.GetAllCoins(ctx)
	if err != nil {
		t.Fatalf("failed to get coins: %v", err)
	}

	p, err := s.CreatePortfolio(ctx, "Test Portfolio")
	if err != nil {
		t.Fatalf("failed to create portfolio: %v", err)
	}

	// Add holdings with amount=1 each
	for _, c := range storedCoins {
		if err := s.UpsertHolding(ctx, p.ID, c.ID, 1.0); err != nil {
			t.Fatalf("failed to upsert holding: %v", err)
		}
	}

	holdings, err := s.GetHoldingsForPortfolio(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get holdings: %v", err)
	}

	if len(holdings) != 2 {
		t.Fatalf("expected 2 holdings, got %d", len(holdings))
	}

	// proportions should be 25% and 75%
	if holdings[0].Proportion != 75.0 {
		t.Errorf("expected first holding proportion 75.0, got %f", holdings[0].Proportion)
	}
	if holdings[1].Proportion != 25.0 {
		t.Errorf("expected second holding proportion 25.0, got %f", holdings[1].Proportion)
	}
}

func TestGetHoldingsProportionZeroWhenNoValue(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Create coin with rate=0
	coin := Coin{
		ApiID:  "zerocoin",
		Name:   "Zero Coin",
		Ticker: "ZERO",
		Rate:   0,
	}
	if err := s.UpsertCoin(ctx, coin); err != nil {
		t.Fatalf("failed to upsert coin: %v", err)
	}

	coins, err := s.GetAllCoins(ctx)
	if err != nil {
		t.Fatalf("failed to get coins: %v", err)
	}
	coinID := coins[0].ID

	p, err := s.CreatePortfolio(ctx, "Test Portfolio")
	if err != nil {
		t.Fatalf("failed to create portfolio: %v", err)
	}

	if err := s.UpsertHolding(ctx, p.ID, coinID, 5.0); err != nil {
		t.Fatalf("failed to upsert holding: %v", err)
	}

	holdings, err := s.GetHoldingsForPortfolio(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get holdings: %v", err)
	}

	if len(holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(holdings))
	}

	if holdings[0].Proportion != 0 {
		t.Errorf("expected proportion 0 when value is 0, got %f", holdings[0].Proportion)
	}
}

func TestDeleteHolding(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Create coin and portfolio
	coin := Coin{
		ApiID:  "bitcoin",
		Name:   "Bitcoin",
		Ticker: "BTC",
		Rate:   67000.00,
	}
	if err := s.UpsertCoin(ctx, coin); err != nil {
		t.Fatalf("failed to upsert coin: %v", err)
	}

	coins, err := s.GetAllCoins(ctx)
	if err != nil {
		t.Fatalf("failed to get coins: %v", err)
	}
	coinID := coins[0].ID

	p, err := s.CreatePortfolio(ctx, "Test Portfolio")
	if err != nil {
		t.Fatalf("failed to create portfolio: %v", err)
	}

	// Add holding
	if err := s.UpsertHolding(ctx, p.ID, coinID, 1.5); err != nil {
		t.Fatalf("failed to upsert holding: %v", err)
	}

	holdings, err := s.GetHoldingsForPortfolio(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get holdings: %v", err)
	}

	if len(holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(holdings))
	}

	holdingID := holdings[0].ID

	// Delete holding
	if err := s.DeleteHolding(ctx, holdingID); err != nil {
		t.Fatalf("failed to delete holding: %v", err)
	}

	// Verify it's gone
	holdings, err = s.GetHoldingsForPortfolio(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get holdings after delete: %v", err)
	}

	if len(holdings) != 0 {
		t.Errorf("expected 0 holdings after delete, got %d", len(holdings))
	}
}

func TestDeleteHoldingNonExistentIsNoOp(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Delete non-existent ID - should not error
	if err := s.DeleteHolding(ctx, 99999); err != nil {
		t.Errorf("expected no error deleting non-existent holding, got: %v", err)
	}
}

func TestRenamePortfolio(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Create a portfolio
	p, err := s.CreatePortfolio(ctx, "Old Name")
	if err != nil {
		t.Fatalf("failed to create portfolio: %v", err)
	}

	// Rename it
	if err := s.RenamePortfolio(ctx, p.ID, "New Name"); err != nil {
		t.Fatalf("failed to rename portfolio: %v", err)
	}

	// Verify the rename
	portfolios, err := s.GetAllPortfolios(ctx)
	if err != nil {
		t.Fatalf("failed to get portfolios: %v", err)
	}

	if len(portfolios) != 1 {
		t.Fatalf("expected 1 portfolio, got %d", len(portfolios))
	}

	if portfolios[0].Name != "New Name" {
		t.Errorf("expected name 'New Name', got %s", portfolios[0].Name)
	}
}

func TestRenamePortfolioDuplicateName(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Create two portfolios
	p1, err := s.CreatePortfolio(ctx, "Portfolio A")
	if err != nil {
		t.Fatalf("failed to create first portfolio: %v", err)
	}

	_, err = s.CreatePortfolio(ctx, "Portfolio B")
	if err != nil {
		t.Fatalf("failed to create second portfolio: %v", err)
	}

	// Try to rename p1 to "Portfolio B" - should fail due to UNIQUE constraint
	if err := s.RenamePortfolio(ctx, p1.ID, "Portfolio B"); err == nil {
		t.Error("expected error when renaming to duplicate name, got nil")
	}
}

func TestRenamePortfolioNotFound(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Try to rename non-existent portfolio
	if err := s.RenamePortfolio(ctx, 99999, "New Name"); err == nil {
		t.Error("expected error when renaming non-existent portfolio, got nil")
	}
}

func TestDeletePortfolio(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Create a portfolio
	p, err := s.CreatePortfolio(ctx, "To Delete")
	if err != nil {
		t.Fatalf("failed to create portfolio: %v", err)
	}

	// Delete it
	if err := s.DeletePortfolio(ctx, p.ID); err != nil {
		t.Fatalf("failed to delete portfolio: %v", err)
	}

	// Verify it's gone
	portfolios, err := s.GetAllPortfolios(ctx)
	if err != nil {
		t.Fatalf("failed to get portfolios: %v", err)
	}

	if len(portfolios) != 0 {
		t.Errorf("expected 0 portfolios after delete, got %d", len(portfolios))
	}
}

func TestDeletePortfolioCascadeHoldings(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Create a coin
	coin := Coin{
		ApiID:  "bitcoin",
		Name:   "Bitcoin",
		Ticker: "BTC",
		Rate:   67000.00,
	}
	if err := s.UpsertCoin(ctx, coin); err != nil {
		t.Fatalf("failed to upsert coin: %v", err)
	}

	coins, err := s.GetAllCoins(ctx)
	if err != nil {
		t.Fatalf("failed to get coins: %v", err)
	}
	coinID := coins[0].ID

	// Create a portfolio
	p, err := s.CreatePortfolio(ctx, "Test Portfolio")
	if err != nil {
		t.Fatalf("failed to create portfolio: %v", err)
	}

	// Add a holding
	if err := s.UpsertHolding(ctx, p.ID, coinID, 1.5); err != nil {
		t.Fatalf("failed to upsert holding: %v", err)
	}

	// Verify holding exists
	holdings, err := s.GetHoldingsForPortfolio(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get holdings: %v", err)
	}
	if len(holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(holdings))
	}

	// Delete the portfolio
	if err := s.DeletePortfolio(ctx, p.ID); err != nil {
		t.Fatalf("failed to delete portfolio: %v", err)
	}

	// Verify holdings are cascade-deleted
	holdings, err = s.GetHoldingsForPortfolio(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get holdings after delete: %v", err)
	}
	if len(holdings) != 0 {
		t.Errorf("expected 0 holdings after portfolio delete, got %d", len(holdings))
	}
}

func TestDeletePortfolioNotFound(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Delete non-existent portfolio - should not error (idempotent)
	if err := s.DeletePortfolio(ctx, 99999); err != nil {
		t.Errorf("expected no error deleting non-existent portfolio, got: %v", err)
	}
}

func TestCreatePortfolioDuplicateName(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	s := NewSQLiteStore(database)
	defer func() {
		_ = s.Close()
	}()

	// Create first portfolio
	if _, err := s.CreatePortfolio(ctx, "Duplicate Name"); err != nil {
		t.Fatalf("failed to create first portfolio: %v", err)
	}

	// Try to create second with same name - should fail due to UNIQUE constraint
	if _, err := s.CreatePortfolio(ctx, "Duplicate Name"); err == nil {
		t.Error("expected error when creating portfolio with duplicate name, got nil")
	}
}
