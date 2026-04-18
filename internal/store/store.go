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

// Holding represents a coin position in a portfolio (raw row, no joins).
type Holding struct {
	ID          int64
	PortfolioID int64
	CoinID      int64
	Amount      float64
}

// HoldingRow is a holding joined with coin data and with computed portfolio metrics.
type HoldingRow struct {
	ID          int64
	PortfolioID int64
	CoinID      int64
	ApiID       string
	Name        string
	Ticker      string
	Amount      float64
	Rate        float64
	PriceChange float64
	Value       float64 // Amount * Rate, computed in SQL
	Proportion  float64 // Value / portfolio_total * 100, computed via window function
}

// Currency represents a fiat currency with its code and display name.
type Currency struct {
	Code string
	Name string
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

	// new in Slice 7
	UpsertHolding(ctx context.Context, portfolioID, coinID int64, amount float64) error
	DeleteHolding(ctx context.Context, id int64) error
	GetHoldingsForPortfolio(ctx context.Context, portfolioID int64) ([]HoldingRow, error)

	// new in Slice 9
	RenamePortfolio(ctx context.Context, id int64, name string) error
	DeletePortfolio(ctx context.Context, id int64) error

	// new in Slice 13
	UpsertCurrencies(ctx context.Context, currencies []Currency) error
	GetAllCurrencies(ctx context.Context) ([]Currency, error)
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
}
