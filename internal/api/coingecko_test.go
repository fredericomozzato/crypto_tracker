package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

	coins, err := client.FetchMarkets(context.Background(), "usd", 1)
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

	_, err := client.FetchMarkets(context.Background(), "usd", 1)
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

	_, err := client.FetchMarkets(context.Background(), "usd", 1)
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

	_, err := client.FetchMarkets(ctx, "usd", 1)
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

	_, err := client.FetchMarkets(context.Background(), "usd", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Ensure HTTPClient implements CoinGeckoClient interface
var _ CoinGeckoClient = (*HTTPClient)(nil)

// FetchPrices tests

func TestFetchPricesSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		if r.URL.Query().Get("ids") != "bitcoin,ethereum" {
			t.Errorf("expected ids=bitcoin,ethereum, got %s", r.URL.Query().Get("ids"))
		}
		if r.URL.Query().Get("vs_currencies") != "usd" {
			t.Errorf("expected vs_currencies=usd, got %s", r.URL.Query().Get("vs_currencies"))
		}

		response := map[string]interface{}{
			"bitcoin": map[string]interface{}{
				"usd": 67000.00,
			},
			"ethereum": map[string]interface{}{
				"usd": 3500.00,
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

	prices, err := client.FetchPrices(context.Background(), []string{"bitcoin", "ethereum"}, "usd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(prices) != 2 {
		t.Fatalf("expected 2 prices, got %d", len(prices))
	}

	if prices["bitcoin"] != 67000.00 {
		t.Errorf("expected bitcoin price 67000.00, got %f", prices["bitcoin"])
	}

	if prices["ethereum"] != 3500.00 {
		t.Errorf("expected ethereum price 3500.00, got %f", prices["ethereum"])
	}
}

func TestFetchPricesEmptyIDs(t *testing.T) {
	client := &HTTPClient{
		httpClient: &http.Client{},
		baseURL:    "http://localhost",
	}

	prices, err := client.FetchPrices(context.Background(), []string{}, "usd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(prices) != 0 {
		t.Errorf("expected empty prices, got %d", len(prices))
	}
}

func TestFetchPricesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limit exceeded"))
	}))
	defer server.Close()

	client := &HTTPClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	_, err := client.FetchPrices(context.Background(), []string{"bitcoin"}, "usd")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "429") {
		t.Errorf("expected error to contain status code 429, got: %s", err.Error())
	}
}

func TestFetchPricesContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client := &HTTPClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.FetchPrices(ctx, []string{"bitcoin"}, "usd")
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

func TestFetchPricesWithAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("x-cg-demo-api-key")
		if apiKey != "test-key" {
			t.Errorf("expected API key 'test-key', got %s", apiKey)
		}

		response := map[string]interface{}{
			"bitcoin": map[string]interface{}{
				"usd": 67000.00,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &HTTPClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
		apiKey:     "test-key",
	}

	prices, err := client.FetchPrices(context.Background(), []string{"bitcoin"}, "usd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if prices["bitcoin"] != 67000.00 {
		t.Errorf("expected bitcoin price 67000.00, got %f", prices["bitcoin"])
	}
}

// Rate limiting tests

func TestThrottleEnforcesMinimumInterval(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		response := []map[string]interface{}{
			{
				"id":            "bitcoin",
				"symbol":        "btc",
				"name":          "Bitcoin",
				"current_price": 67000.00,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	minInterval := 50 * time.Millisecond
	client := newHTTPClientWithInterval("", minInterval, server.URL)

	// First request
	start := time.Now()
	_, err := client.FetchMarkets(context.Background(), "usd", 1)
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}

	// Second request should be throttled
	_, err = client.FetchMarkets(context.Background(), "usd", 1)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}

	if elapsed < minInterval {
		t.Errorf("expected second call to be throttled for at least %v, but took %v", minInterval, elapsed)
	}

	if requestCount != 2 {
		t.Errorf("expected 2 requests, got %d", requestCount)
	}
}

func TestThrottleWithContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := []map[string]interface{}{}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	minInterval := 100 * time.Millisecond
	client := newHTTPClientWithInterval("", minInterval, server.URL)

	// First request to set lastRequestAt
	_, _ = client.FetchMarkets(context.Background(), "usd", 1)

	// Second request with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := client.FetchMarkets(ctx, "usd", 1)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context canceled error, got: %s", err.Error())
	}
}

func TestFetchMarketsRateLimit429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limit exceeded"))
	}))
	defer server.Close()

	client := newHTTPClientWithInterval("", 10*time.Millisecond, server.URL)

	_, err := client.FetchMarkets(context.Background(), "usd", 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsRateLimitError(err) {
		t.Errorf("expected IsRateLimitError to return true, got false")
	}

	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected error to be *RateLimitError, got %T", err)
	}

	if !strings.Contains(rle.Body, "rate limit exceeded") {
		t.Errorf("expected Body to contain 'rate limit exceeded', got %q", rle.Body)
	}
}

func TestFetchPricesRateLimit429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("API rate limit exceeded"))
	}))
	defer server.Close()

	client := newHTTPClientWithInterval("", 10*time.Millisecond, server.URL)

	_, err := client.FetchPrices(context.Background(), []string{"bitcoin"}, "usd")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsRateLimitError(err) {
		t.Errorf("expected IsRateLimitError to return true, got false")
	}

	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected error to be *RateLimitError, got %T", err)
	}

	if !strings.Contains(rle.Body, "API rate limit exceeded") {
		t.Errorf("expected Body to contain 'API rate limit exceeded', got %q", rle.Body)
	}
}

func TestFetchMarkets429WithRetryAfterHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limit exceeded"))
	}))
	defer server.Close()

	client := newHTTPClientWithInterval("", 10*time.Millisecond, server.URL)

	_, err := client.FetchMarkets(context.Background(), "usd", 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected error to be *RateLimitError, got %T", err)
	}

	if rle.RetryAfter != 30*time.Second {
		t.Errorf("expected RetryAfter 30s, got %v", rle.RetryAfter)
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "RateLimitError returns true",
			err:      &RateLimitError{Body: "rate limited"},
			expected: true,
		},
		{
			name:     "Regular error returns false",
			err:      errors.New("network error"),
			expected: false,
		},
		{
			name:     "Nil error returns false",
			err:      nil,
			expected: false,
		},
		{
			name:     "Wrapped RateLimitError returns true",
			err:      fmt.Errorf("wrapped: %w", &RateLimitError{Body: "rate limited"}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRateLimitError(tt.err)
			if got != tt.expected {
				t.Errorf("IsRateLimitError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRetryAfterFromError(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		defaultDuration time.Duration
		expected        time.Duration
	}{
		{
			name:            "RateLimitError with RetryAfter set",
			err:             &RateLimitError{Body: "rate limited", RetryAfter: 30 * time.Second},
			defaultDuration: 60 * time.Second,
			expected:        30 * time.Second,
		},
		{
			name:            "RateLimitError with zero RetryAfter uses default",
			err:             &RateLimitError{Body: "rate limited"},
			defaultDuration: 60 * time.Second,
			expected:        60 * time.Second,
		},
		{
			name:            "Non-RateLimitError returns default",
			err:             errors.New("network error"),
			defaultDuration: 60 * time.Second,
			expected:        60 * time.Second,
		},
		{
			name:            "Nil error returns default",
			err:             nil,
			defaultDuration: 60 * time.Second,
			expected:        60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RetryAfterFromError(tt.err, tt.defaultDuration)
			if got != tt.expected {
				t.Errorf("RetryAfterFromError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewHTTPClientDefaultMinInterval(t *testing.T) {
	client := NewHTTPClient("")

	if client.minInterval != 2*time.Second {
		t.Errorf("expected minInterval to be 2s, got %v", client.minInterval)
	}
}

func TestFetchSupportedCurrenciesSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := []string{"usd", "eur", "btc"}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &HTTPClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	codes, err := client.FetchSupportedCurrencies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(codes) != 3 {
		t.Fatalf("expected 3 codes, got %d", len(codes))
	}

	expected := []string{"usd", "eur", "btc"}
	for i, exp := range expected {
		if codes[i] != exp {
			t.Errorf("position %d: expected %s, got %s", i, exp, codes[i])
		}
	}
}

func TestFetchSupportedCurrenciesWithAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("x-cg-demo-api-key")
		if apiKey != "test-key" {
			t.Errorf("expected API key 'test-key', got %s", apiKey)
		}

		response := []string{"usd"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &HTTPClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
		apiKey:     "test-key",
	}

	_, err := client.FetchSupportedCurrencies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchSupportedCurrencies429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limit exceeded"))
	}))
	defer server.Close()

	client := newHTTPClientWithInterval("", 10*time.Millisecond, server.URL)

	_, err := client.FetchSupportedCurrencies(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsRateLimitError(err) {
		t.Errorf("expected IsRateLimitError to return true, got false")
	}
}

func TestFetchSupportedCurrenciesNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	_, err := client.FetchSupportedCurrencies(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFetchSupportedCurrenciesContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client := &HTTPClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.FetchSupportedCurrencies(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}
