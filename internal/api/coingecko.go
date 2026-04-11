package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

// CoinGeckoClient defines the interface for fetching cryptocurrency market data.
type CoinGeckoClient interface {
	FetchMarkets(ctx context.Context, limit int) ([]store.Coin, error)
	FetchPrices(ctx context.Context, apiIDs []string) (map[string]float64, error)
}

// HTTPClient implements CoinGeckoClient using HTTP requests.
type HTTPClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
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
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    baseURL,
		apiKey:     apiKey,
	}
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
	params := url.Values{}
	params.Set("vs_currency", "usd")
	params.Set("order", "market_cap_desc")
	params.Set("per_page", strconv.Itoa(limit))
	params.Set("page", "1")
	params.Set("sparkline", "false")
	params.Set("price_change_percentage", "24h")

	u := c.baseURL + "/coins/markets?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
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
	if len(apiIDs) == 0 {
		return make(map[string]float64), nil
	}

	params := url.Values{}
	params.Set("ids", strings.Join(apiIDs, ","))
	params.Set("vs_currencies", "usd")

	u := c.baseURL + "/simple/price?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
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
