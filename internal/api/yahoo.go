package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/dperkins-ct/random-walk/internal/models"
)

const yahooChartBase = "https://query1.finance.yahoo.com/v8/finance/chart/"

// maxHistoryYears is the number of years always fetched from Yahoo Finance.
// The local cache stores the full window so any --period flag can be served
// without re-fetching. filterByPeriod in the analysis handler trims to the
// requested window.
const maxHistoryYears = 5

// YahooHandler fetches historical price data from Yahoo Finance.
// No API key is required.
type YahooHandler struct {
	httpClient *http.Client
}

// NewYahooHandler constructs a new YahooHandler.
func NewYahooHandler() *YahooHandler {
	return &YahooHandler{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchPrices retrieves up to maxHistoryYears of daily OHLCV + adjusted close
// for ticker. The caller should use filterByPeriod to trim to the desired window.
func (h *YahooHandler) FetchPrices(ticker string) ([]models.DailyPrice, error) {
	now := time.Now()
	start := now.AddDate(-maxHistoryYears, 0, 0)

	params := url.Values{}
	params.Set("period1", strconv.FormatInt(start.Unix(), 10))
	params.Set("period2", strconv.FormatInt(now.Unix(), 10))
	params.Set("interval", "1d")
	params.Set("events", "adjclose")
	reqURL := yahooChartBase + ticker + "?" + params.Encode()

	body, err := h.get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("yahoo fetch prices %s: %w", ticker, err)
	}
	return parseYahooChart(ticker, body)
}

// get performs an HTTP GET with browser-like headers that Yahoo Finance requires.
func (h *YahooHandler) get(reqURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 "+
			"(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json,text/plain,*/*")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from Yahoo Finance", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// parseYahooChart deserialises the v8/finance/chart JSON response.
// Null entries (market holidays, data gaps) are skipped automatically.
func parseYahooChart(ticker string, body []byte) ([]models.DailyPrice, error) {
	var raw struct {
		Chart struct {
			Result []struct {
				Timestamp  []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Open   []*float64 `json:"open"`
						High   []*float64 `json:"high"`
						Low    []*float64 `json:"low"`
						Close  []*float64 `json:"close"`
						Volume []*float64 `json:"volume"`
					} `json:"quote"`
					AdjClose []struct {
						AdjClose []*float64 `json:"adjclose"`
					} `json:"adjclose"`
				} `json:"indicators"`
			} `json:"result"`
			Error *struct {
				Code        string `json:"code"`
				Description string `json:"description"`
			} `json:"error"`
		} `json:"chart"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse chart JSON for %s: %w", ticker, err)
	}
	if raw.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo chart error for %s: %s - %s",
			ticker, raw.Chart.Error.Code, raw.Chart.Error.Description)
	}
	if len(raw.Chart.Result) == 0 {
		return nil, fmt.Errorf("no chart data returned for %s", ticker)
	}

	res := raw.Chart.Result[0]
	if len(res.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no quote indicators for %s", ticker)
	}
	q := res.Indicators.Quote[0]

	var adjSlice []*float64
	if len(res.Indicators.AdjClose) > 0 {
		adjSlice = res.Indicators.AdjClose[0].AdjClose
	}

	prices := make([]models.DailyPrice, 0, len(res.Timestamp))
	for i, ts := range res.Timestamp {
		if i >= len(q.Close) || q.Close[i] == nil {
			continue
		}
		p := models.DailyPrice{
			Date:  time.Unix(ts, 0).UTC().Format("2006-01-02"),
			Close: *q.Close[i],
		}
		p.AdjClose = p.Close
		if adjSlice != nil && i < len(adjSlice) && adjSlice[i] != nil {
			p.AdjClose = *adjSlice[i]
		}
		if i < len(q.Open) && q.Open[i] != nil {
			p.Open = *q.Open[i]
		}
		if i < len(q.High) && q.High[i] != nil {
			p.High = *q.High[i]
		}
		if i < len(q.Low) && q.Low[i] != nil {
			p.Low = *q.Low[i]
		}
		if i < len(q.Volume) && q.Volume[i] != nil {
			p.Volume = int64(*q.Volume[i])
		}
		prices = append(prices, p)
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("no valid price rows for %s after filtering nulls", ticker)
	}

	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Date < prices[j].Date
	})
	return prices, nil
}
