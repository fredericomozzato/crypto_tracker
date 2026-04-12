package store

import "context"

// Coin represents a cryptocurrency with its market data.
type Coin struct {
	ID          int64
	ApiID       string
	Name        string
	Ticker      string
	Rate        float64
	PriceChange float64
	MarketRank  int
	UpdatedAt   int64
}

// Portfolio represents a named collection of holdings.
type Portfolio struct {
	ID        int64
	Name      string
	CreatedAt int64
}

// Store defines the interface for cryptocurrency data persistence.
type Store interface {
	UpsertCoin(ctx context.Context, c Coin) error
	GetAllCoins(ctx context.Context) ([]Coin, error)
	UpdatePrices(ctx context.Context, prices map[string]float64) error
	Close() error

	// new in Slice 6
	CreatePortfolio(ctx context.Context, name string) (Portfolio, error)
	GetAllPortfolios(ctx context.Context) ([]Portfolio, error)
}
