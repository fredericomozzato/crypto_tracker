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
