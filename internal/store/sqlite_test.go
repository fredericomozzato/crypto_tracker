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
	database, err := db.Open(dbPath)
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
	database, err := db.Open(dbPath)
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
	database, err := db.Open(dbPath)
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
	database, err := db.Open(dbPath)
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
