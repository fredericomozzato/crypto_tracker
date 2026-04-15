package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

// DefaultRetryAfter is the default cooldown duration when rate limited.
const DefaultRetryAfter = 60 * time.Second

// RateLimitError is returned when the CoinGecko API responds with HTTP 429.
type RateLimitError struct {
	Body       string
	RetryAfter time.Duration // cooldown duration; 0 means use a default
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited: 429 %s", e.Body)
}

// IsRateLimitError reports whether err is a *RateLimitError.
func IsRateLimitError(err error) bool {
	var rle *RateLimitError
	return errors.As(err, &rle)
}

// RetryAfterFromError extracts the RetryAfter from a RateLimitError.
// Returns defaultDuration if err is not a RateLimitError or has RetryAfter == 0.
func RetryAfterFromError(err error, defaultDuration time.Duration) time.Duration {
	var rle *RateLimitError
	if errors.As(err, &rle) && rle.RetryAfter > 0 {
		return rle.RetryAfter
	}
	return defaultDuration
}

// CoinGeckoClient defines the interface for fetching cryptocurrency market data.
type CoinGeckoClient interface {
	FetchMarkets(ctx context.Context, limit int) ([]store.Coin, error)
	FetchPrices(ctx context.Context, apiIDs []string) (map[string]float64, error)
}

// HTTPClient implements CoinGeckoClient using HTTP requests.
type HTTPClient struct {
	httpClient    *http.Client
	baseURL       string
	apiKey        string
	mu            sync.Mutex
	lastRequestAt time.Time
	minInterval   time.Duration // minimum gap between API calls (default 2s)
}

// NewHTTPClient creates a new HTTP client for CoinGecko API.
// If apiKey is empty, uses the free tier endpoint.
// If apiKey is provided, uses the pro endpoint and sets the API key header.
func NewHTTPClient(apiKey string) *HTTPClient {
	baseURL := "https://api.coingecko.com/api/v3"
	if apiKey != "" {
		baseURL = "https://pro-api.coingecko.com/api/v3"
	}

	return &HTTPClient{
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		baseURL:     baseURL,
		apiKey:      apiKey,
		minInterval: 2 * time.Second,
	}
}

// newHTTPClientWithInterval creates an HTTPClient with a custom minimum request interval.
// Used for testing; production code should use NewHTTPClient.
func newHTTPClientWithInterval(apiKey string, interval time.Duration, serverURL string) *HTTPClient {
	return &HTTPClient{
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		baseURL:     serverURL,
		apiKey:      apiKey,
		minInterval: interval,
	}
}

// throttle sleeps until minInterval has elapsed since lastRequestAt.
// Respects context cancellation.
func (c *HTTPClient) throttle(ctx context.Context) error {
	c.mu.Lock()
	elapsed := time.Since(c.lastRequestAt)
	if elapsed < c.minInterval {
		sleepDuration := c.minInterval - elapsed
		c.mu.Unlock()

		timer := time.NewTimer(sleepDuration)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			c.mu.Lock()
			c.lastRequestAt = time.Now()
			c.mu.Unlock()
			return nil
		}
	}
	c.lastRequestAt = time.Now()
	c.mu.Unlock()
	return nil
}

// coinGeckoMarketResponse represents a single item from the /coins/markets endpoint.
type coinGeckoMarketResponse struct {
	ID                       string  `json:"id"`
	Symbol                   string  `json:"symbol"`
	Name                     string  `json:"name"`
	CurrentPrice             float64 `json:"current_price"`
	PriceChangePercentage24h float64 `json:"price_change_percentage_24h"`
	MarketCapRank            int     `json:"market_cap_rank"`
}

// FetchMarkets fetches market data for the top cryptocurrencies.
func (c *HTTPClient) FetchMarkets(ctx context.Context, limit int) ([]store.Coin, error) {
	if err := c.throttle(ctx); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("vs_currency", "usd")
	params.Set("order", "market_cap_desc")
	params.Set("per_page", strconv.Itoa(limit))
	params.Set("page", "1")
	params.Set("sparkline", "false")
	params.Set("price_change_percentage", "24h")

	u := c.baseURL + "/coins/markets?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("x-cg-demo-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching markets: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusTooManyRequests {
		body, _ := io.ReadAll(resp.Body)
		rle := &RateLimitError{
			Body: string(body),
		}
		if retryAfterHeader := resp.Header.Get("Retry-After"); retryAfterHeader != "" {
			if seconds, err := strconv.Atoi(retryAfterHeader); err == nil {
				rle.RetryAfter = time.Duration(seconds) * time.Second
			}
		}
		return nil, rle
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("fetching markets: %d (failed to read response body: %w)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("fetching markets: %d %s", resp.StatusCode, string(body))
	}

	var apiResp []coinGeckoMarketResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	coins := make([]store.Coin, len(apiResp))
	for i, item := range apiResp {
		coins[i] = store.Coin{
			ApiID:       item.ID,
			Name:        item.Name,
			Ticker:      strings.ToUpper(item.Symbol),
			Rate:        item.CurrentPrice,
			PriceChange: item.PriceChangePercentage24h,
			MarketRank:  item.MarketCapRank,
		}
	}

	return coins, nil
}

// coinGeckoPriceResponse represents the response from the /simple/price endpoint.
type coinGeckoPriceResponse map[string]map[string]float64

// FetchPrices fetches current prices for the given coin API IDs.
// Returns a map of api_id -> USD price.
func (c *HTTPClient) FetchPrices(ctx context.Context, apiIDs []string) (map[string]float64, error) {
	if err := c.throttle(ctx); err != nil {
		return nil, err
	}

	if len(apiIDs) == 0 {
		return make(map[string]float64), nil
	}

	params := url.Values{}
	params.Set("ids", strings.Join(apiIDs, ","))
	params.Set("vs_currencies", "usd")

	u := c.baseURL + "/simple/price?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("x-cg-demo-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching prices: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusTooManyRequests {
		body, _ := io.ReadAll(resp.Body)
		rle := &RateLimitError{
			Body: string(body),
		}
		if retryAfterHeader := resp.Header.Get("Retry-After"); retryAfterHeader != "" {
			if seconds, err := strconv.Atoi(retryAfterHeader); err == nil {
				rle.RetryAfter = time.Duration(seconds) * time.Second
			}
		}
		return nil, rle
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("fetching prices: %d (failed to read response body: %w)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("fetching prices: %d %s", resp.StatusCode, string(body))
	}

	var apiResp coinGeckoPriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	prices := make(map[string]float64, len(apiResp))
	for apiID, priceData := range apiResp {
		if usdPrice, ok := priceData["usd"]; ok {
			prices[apiID] = usdPrice
		}
	}

	return prices, nil
}
