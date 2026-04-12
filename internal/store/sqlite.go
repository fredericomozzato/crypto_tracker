package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SQLiteStore implements the Store interface using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLiteStore with the given database handle.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// UpsertCoin inserts a new coin or updates an existing one based on api_id.
// The updated_at field is set to the current timestamp.
func (s *SQLiteStore) UpsertCoin(ctx context.Context, c Coin) error {
	now := time.Now().Unix()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO coins (api_id, name, ticker, rate, price_change, market_rank, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(api_id) DO UPDATE SET
			name = excluded.name,
			ticker = excluded.ticker,
			rate = excluded.rate,
			price_change = excluded.price_change,
			market_rank = excluded.market_rank,
			updated_at = excluded.updated_at
	`, c.ApiID, c.Name, c.Ticker, c.Rate, c.PriceChange, c.MarketRank, now)
	if err != nil {
		return fmt.Errorf("upserting coin %s: %w", c.ApiID, err)
	}

	return nil
}

// GetAllCoins returns all coins ordered by market_rank ascending.
func (s *SQLiteStore) GetAllCoins(ctx context.Context) ([]Coin, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, api_id, name, ticker, rate, price_change, market_rank, updated_at
		FROM coins
		ORDER BY market_rank ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying coins: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	coins := make([]Coin, 0)
	for rows.Next() {
		var c Coin
		if err := rows.Scan(&c.ID, &c.ApiID, &c.Name, &c.Ticker, &c.Rate, &c.PriceChange, &c.MarketRank, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning coin: %w", err)
		}
		coins = append(coins, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating coins: %w", err)
	}

	return coins, nil
}

// UpdatePrices updates the rate and updated_at for coins matching the given api_ids.
// Unknown api_ids are silently ignored. All updates happen in a single transaction.
func (s *SQLiteStore) UpdatePrices(ctx context.Context, prices map[string]float64) error {
	if len(prices) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx, `
		UPDATE coins
		SET rate = ?, updated_at = ?
		WHERE api_id = ?
	`)
	if err != nil {
		return fmt.Errorf("preparing update statement: %w", err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	now := time.Now().Unix()
	for apiID, price := range prices {
		if _, err := stmt.ExecContext(ctx, price, now, apiID); err != nil {
			return fmt.Errorf("updating price for %s: %w", apiID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// CreatePortfolio inserts a new portfolio and returns it with the generated ID.
func (s *SQLiteStore) CreatePortfolio(ctx context.Context, name string) (Portfolio, error) {
	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO portfolios (name, created_at) VALUES (?, ?)
	`, name, now)
	if err != nil {
		return Portfolio{}, fmt.Errorf("creating portfolio %q: %w", name, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Portfolio{}, fmt.Errorf("getting portfolio id: %w", err)
	}
	return Portfolio{ID: id, Name: name, CreatedAt: now}, nil
}

// GetAllPortfolios returns all portfolios ordered by created_at ascending.
func (s *SQLiteStore) GetAllPortfolios(ctx context.Context) ([]Portfolio, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, created_at FROM portfolios ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying portfolios: %w", err)
	}
	defer func() { _ = rows.Close() }()

	portfolios := make([]Portfolio, 0)
	for rows.Next() {
		var p Portfolio
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning portfolio: %w", err)
		}
		portfolios = append(portfolios, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating portfolios: %w", err)
	}
	return portfolios, nil
}
