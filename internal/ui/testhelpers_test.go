package ui

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

// StubStore implements store.Store for testing
type StubStore struct {
	coins       []store.Coin
	portfolios  []store.Portfolio
	holdingRows []store.HoldingRow
	err         error
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

func (s *StubStore) CreatePortfolio(ctx context.Context, name string) (store.Portfolio, error) {
	if s.err != nil {
		return store.Portfolio{}, s.err
	}
	p := store.Portfolio{
		ID:        int64(len(s.portfolios) + 1),
		Name:      name,
		CreatedAt: time.Now().Unix(),
	}
	s.portfolios = append(s.portfolios, p)
	return p, nil
}

func (s *StubStore) GetAllPortfolios(ctx context.Context) ([]store.Portfolio, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.portfolios, nil
}

func (s *StubStore) UpsertHolding(ctx context.Context, portfolioID, coinID int64, amount float64) error {
	return s.err
}

func (s *StubStore) DeleteHolding(ctx context.Context, id int64) error {
	return s.err
}

func (s *StubStore) GetHoldingsForPortfolio(ctx context.Context, portfolioID int64) ([]store.HoldingRow, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.holdingRows, nil
}

// StubAPI implements api.CoinGeckoClient for testing
type StubAPI struct {
	coins             []store.Coin
	prices            map[string]float64
	err               error
	fetchMarketsCalls []int
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

func threeCoins() []store.Coin {
	return makeCoins(3)
}

func makeCoins(n int) []store.Coin {
	coins := make([]store.Coin, n)
	for i := range coins {
		coins[i] = store.Coin{
			ID:         int64(i + 1),
			ApiID:      fmt.Sprintf("coin-%d", i+1),
			Name:       fmt.Sprintf("Coin %d", i+1),
			Ticker:     fmt.Sprintf("C%d", i+1),
			Rate:       float64((i + 1) * 100),
			MarketRank: i + 1,
		}
	}
	return coins
}

var testCtx = context.Background()

// setupMarketsModel creates a MarketsModel with pre-loaded coins for cursor tests.
func setupMarketsModel(t *testing.T, coins []store.Coin) MarketsModel {
	t.Helper()
	m := NewMarketsModel(testCtx, &StubStore{coins: coins}, &StubAPI{})
	m.width = 120
	m.height = 40
	updated, _ := m.update(coinsLoadedMsg{coins: coins})
	return updated
}
