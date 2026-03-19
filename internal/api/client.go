// Package api provides a client for fetching stock data from Alpha Vantage.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"
)

const (
	baseURL       = "https://www.alphavantage.co/query"
	defaultTimeout = 15 * time.Second
)

// DailyPrice holds OHLCV data for a single trading day.
type DailyPrice struct {
	Date   string
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

// Client is an Alpha Vantage REST client.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new API client using the provided API key.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// alphavantageResponse is the raw JSON envelope returned by Alpha Vantage
// for the TIME_SERIES_DAILY function.
type alphavantageResponse struct {
	MetaData   map[string]string            `json:"Meta Data"`
	TimeSeries map[string]map[string]string `json:"Time Series (Daily)"`
	Note       string                        `json:"Note"`
	Information string                       `json:"Information"`
}

// GetDailyPrices fetches daily adjusted close prices for the given ticker
// and returns them sorted from oldest to newest.
func (c *Client) GetDailyPrices(ticker string) ([]DailyPrice, error) {
	return c.GetDailyPricesContext(context.Background(), ticker)
}

// GetDailyPricesContext is like GetDailyPrices but accepts a context for
// timeout and cancellation control.
func (c *Client) GetDailyPricesContext(ctx context.Context, ticker string) ([]DailyPrice, error) {
	url := fmt.Sprintf(
		"%s?function=TIME_SERIES_DAILY&symbol=%s&outputsize=compact&apikey=%s",
		baseURL, ticker, c.apiKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d: %s", resp.StatusCode, string(body))
	}

	var raw alphavantageResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	if raw.Note != "" {
		return nil, fmt.Errorf("API note (rate limit?): %s", raw.Note)
	}
	if raw.Information != "" {
		return nil, fmt.Errorf("API information: %s", raw.Information)
	}
	if len(raw.TimeSeries) == 0 {
		return nil, fmt.Errorf("no time series data returned for ticker %q", ticker)
	}

	prices := make([]DailyPrice, 0, len(raw.TimeSeries))
	for date, fields := range raw.TimeSeries {
		closeVal, err := strconv.ParseFloat(fields["4. close"], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid close price for date %s: %w", date, err)
		}
		open, _ := strconv.ParseFloat(fields["1. open"], 64)
		high, _ := strconv.ParseFloat(fields["2. high"], 64)
		low, _ := strconv.ParseFloat(fields["3. low"], 64)
		vol, _ := strconv.ParseInt(fields["5. volume"], 10, 64)

		prices = append(prices, DailyPrice{
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closeVal,
			Volume: vol,
		})
	}

	// Sort oldest → newest so analysis windows are consistent.
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Date < prices[j].Date
	})

	return prices, nil
}

// ClosePrices extracts only the closing prices from a slice of DailyPrice.
func ClosePrices(prices []DailyPrice) []float64 {
	out := make([]float64, len(prices))
	for i, p := range prices {
		out[i] = p.Close
	}
	return out
}
