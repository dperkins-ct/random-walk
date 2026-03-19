package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/dperkins-ct/random-walk/internal/models"
)

const baseURL = "https://www.alphavantage.co/query"

// Handler manages Alpha Vantage API calls.
type Handler struct {
	apiKey     string
	httpClient *http.Client
}

// NewHandler constructs a new API Handler.
func NewHandler(apiKey string) *Handler {
	return &Handler{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchPrices fetches daily adjusted prices for ticker.
// outputSize must be "compact" (last 100 days) or "full" (20+ years).
func (h *Handler) FetchPrices(ticker, outputSize string) ([]models.DailyPrice, error) {
	url := fmt.Sprintf(
		"%s?function=TIME_SERIES_DAILY_ADJUSTED&symbol=%s&outputsize=%s&apikey=%s",
		baseURL, ticker, outputSize, h.apiKey,
	)
	resp, err := h.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch prices for %s: %w", ticker, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response for %s: %w", ticker, err)
	}

	type dailyEntry struct {
		Open     string `json:"1. open"`
		High     string `json:"2. high"`
		Low      string `json:"3. low"`
		Close    string `json:"4. close"`
		AdjClose string `json:"5. adjusted close"`
		Volume   string `json:"6. volume"`
	}
	var raw struct {
		Note            string                `json:"Note"`
		Information     string                `json:"Information"`
		TimeSeriesDaily map[string]dailyEntry `json:"Time Series (Daily)"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse response for %s: %w", ticker, err)
	}
	if raw.Note != "" {
		return nil, fmt.Errorf("alpha vantage rate limit: %s", raw.Note)
	}
	if raw.Information != "" {
		return nil, fmt.Errorf("alpha vantage info: %s", raw.Information)
	}
	if raw.TimeSeriesDaily == nil {
		return nil, fmt.Errorf("no price data returned for %s", ticker)
	}

	prices := make([]models.DailyPrice, 0, len(raw.TimeSeriesDaily))
	for date, vals := range raw.TimeSeriesDaily {
		p := models.DailyPrice{Date: date}
		p.Open, _ = strconv.ParseFloat(vals.Open, 64)
		p.High, _ = strconv.ParseFloat(vals.High, 64)
		p.Low, _ = strconv.ParseFloat(vals.Low, 64)
		p.Close, _ = strconv.ParseFloat(vals.Close, 64)
		p.AdjClose, _ = strconv.ParseFloat(vals.AdjClose, 64)
		v, _ := strconv.ParseInt(vals.Volume, 10, 64)
		p.Volume = v
		prices = append(prices, p)
	}
	// Sort chronologically (oldest first); YYYY-MM-DD sorts correctly lexically.
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Date < prices[j].Date
	})
	return prices, nil
}

// FetchOverview fetches fundamental data for ticker via the OVERVIEW endpoint.
func (h *Handler) FetchOverview(ticker string) (models.Overview, error) {
	url := fmt.Sprintf("%s?function=OVERVIEW&symbol=%s&apikey=%s",
		baseURL, ticker, h.apiKey)
	resp, err := h.httpClient.Get(url)
	if err != nil {
		return models.Overview{}, fmt.Errorf("fetch overview for %s: %w", ticker, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Overview{}, fmt.Errorf("read overview response: %w", err)
	}

	var raw struct {
		Symbol    string `json:"Symbol"`
		Name      string `json:"Name"`
		Sector    string `json:"Sector"`
		PERatio   string `json:"PERatio"`
		ForwardPE string `json:"ForwardPE"`
		Note      string `json:"Note"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return models.Overview{}, fmt.Errorf("parse overview for %s: %w", ticker, err)
	}
	if raw.Note != "" {
		return models.Overview{}, fmt.Errorf("alpha vantage rate limit: %s", raw.Note)
	}
	ov := models.Overview{
		Symbol: raw.Symbol,
		Name:   raw.Name,
		Sector: raw.Sector,
	}
	ov.PERatio, _ = strconv.ParseFloat(raw.PERatio, 64)
	ov.ForwardPE, _ = strconv.ParseFloat(raw.ForwardPE, 64)
	return ov, nil
}
