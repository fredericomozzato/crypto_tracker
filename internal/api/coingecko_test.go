package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

func TestFetchMarketsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		if r.URL.Query().Get("vs_currency") != "usd" {
			t.Errorf("expected vs_currency=usd, got %s", r.URL.Query().Get("vs_currency"))
		}
		if r.URL.Query().Get("order") != "market_cap_desc" {
			t.Errorf("expected order=market_cap_desc, got %s", r.URL.Query().Get("order"))
		}

		response := []map[string]interface{}{
			{
				"id":                          "bitcoin",
				"symbol":                      "btc",
				"name":                        "Bitcoin",
				"current_price":               67000.00,
				"price_change_percentage_24h": -1.23,
				"market_cap_rank":             1,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &HTTPClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	coins, err := client.FetchMarkets(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
}

func TestFetchMarketsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limit exceeded"))
	}))
	defer server.Close()

	client := &HTTPClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	_, err := client.FetchMarkets(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "429") {
		t.Errorf("expected error to contain status code 429, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Errorf("expected error to contain response body, got: %s", err.Error())
	}
}

func TestFetchMarketsNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Immediately close the connection to simulate network error
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("server doesn't support hijacking")
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			t.Fatalf("failed to hijack connection: %v", err)
		}
		_ = conn.Close()
	}))
	defer server.Close()

	client := &HTTPClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	_, err := client.FetchMarkets(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFetchMarketsContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response that will be cancelled
		<-r.Context().Done()
	}))
	defer server.Close()

	client := &HTTPClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.FetchMarkets(ctx, 1)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context canceled error, got: %s", err.Error())
	}
}

func TestNewHTTPClient(t *testing.T) {
	// Test without API key
	client := NewHTTPClient("")
	if client.baseURL != "https://api.coingecko.com/api/v3" {
		t.Errorf("expected base URL for free tier, got %s", client.baseURL)
	}

	// Test with API key
	client = NewHTTPClient("demo-key")
	if client.baseURL != "https://pro-api.coingecko.com/api/v3" {
		t.Errorf("expected base URL for pro tier, got %s", client.baseURL)
	}
	if client.apiKey != "demo-key" {
		t.Errorf("expected API key to be set, got %s", client.apiKey)
	}
}

func TestFetchMarketsWithAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for API key header
		apiKey := r.Header.Get("x-cg-demo-api-key")
		if apiKey != "test-key" {
			t.Errorf("expected API key 'test-key', got %s", apiKey)
		}

		response := []map[string]interface{}{}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &HTTPClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
		apiKey:     "test-key",
	}

	_, err := client.FetchMarkets(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Ensure HTTPClient implements CoinGeckoClient interface
var _ CoinGeckoClient = (*HTTPClient)(nil)

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
