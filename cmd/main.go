package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dperkins-ct/random-walk/internal/analysis"
	"github.com/dperkins-ct/random-walk/internal/api"
	"github.com/dperkins-ct/random-walk/internal/cache"
	"github.com/dperkins-ct/random-walk/internal/indicators"
	"github.com/dperkins-ct/random-walk/internal/output"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run coordinates data fetching, analysis, and output.
// Only main() may call os.Exit; run communicates failures via its error return.
func run(args []string) error {
	_ = loadDotEnv(".env")

	fs := flag.NewFlagSet("random-walk", flag.ContinueOnError)
	apiKey := fs.String("api-key", "",
		"Alpha Vantage API key for P/E + fundamentals.\n"+
			"        Alternatively set ALPHAVANTAGE_API_KEY in your environment or .env file.")
	period := fs.String("period", "1y", "Analysis window: 1y | 2y | 5y")
	riskFreeRate := fs.Float64("risk-free-rate", 0.043, "Annual risk-free rate as decimal (e.g. 0.043)")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		fs.Usage()
		return errors.New("ticker symbol required  e.g.:  random-walk AAPL")
	}
	ticker := fs.Arg(0)
	// Go's flag package stops at the first non-flag arg, so flags that follow
	// the ticker (e.g. ./random-walk AAPL --period 5y) are not parsed on the
	// first pass.  Re-invoke Parse on the remaining args to pick them up.
	if fs.NArg() > 1 {
		_ = fs.Parse(fs.Args()[1:])
	}

	avKey := *apiKey
	if avKey == "" {
		avKey = os.Getenv("ALPHAVANTAGE_API_KEY")
	}

	yh := api.NewYahooHandler()
	// The --market-return flag was removed; CAPM now uses the actual SPY return
	// computed from the fetched market data, which is more accurate.
	analysisHandler := analysis.NewHandler(*riskFreeRate)

	// Prices: always fetch the full 5-year history from Yahoo Finance so that
	// (a) any --period value can be served without re-fetching, and
	// (b) EMA/MACD warmup data is available before the analysis window starts.
	stockPrices, err := fetchPrices(yh, ticker)
	if err != nil {
		return fmt.Errorf("stock data: %w", err)
	}
	spyPrices, err := fetchPrices(yh, "SPY")
	if err != nil {
		return fmt.Errorf("market benchmark (SPY): %w", err)
	}

	// Overview: one Alpha Vantage call, cached for the day.
	var overview indicators.Overview
	if avKey != "" {
		overview, err = fetchAVOverview(api.NewHandler(avKey), ticker)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: fundamentals unavailable for %s: %v\n", ticker, err)
		}
	} else {
		fmt.Fprintf(os.Stderr,
			"info: no API key set -- P/E and sector will be N/A.\n"+
				"      Add ALPHAVANTAGE_API_KEY=<key> to .env for full analysis.\n")
	}

	// Sector ETF prices: fetch once, cached like any other price series.
	// If the sector is unknown or the fetch fails we pass nil and the handler skips it.
	var sectorPrices []indicators.DailyPrice
	if etf, ok := indicators.SectorETF(overview.Sector); ok {
		sectorPrices, err = fetchPrices(yh, etf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: sector ETF (%s) unavailable: %v\n", etf, err)
		}
	}

	result, err := analysisHandler.Analyze(ticker, stockPrices, spyPrices, sectorPrices, overview, *period)
	if err != nil {
		return fmt.Errorf("analysis: %w", err)
	}

	output.Print(result)
	return nil
}

// fetchPrices returns up to 5 years of prices from the local CSV cache if it is
// still fresh (written today), otherwise fetches from Yahoo Finance and
// refreshes the cache.
func fetchPrices(h *api.YahooHandler, ticker string) ([]indicators.DailyPrice, error) {
	cachePath, err := cache.PricesCachePath(ticker)
	if err == nil && cache.IsFresh(cachePath) {
		if prices, err := cache.ReadPrices(ticker); err == nil && len(prices) > 300 {
			// Only use the cache if it has more than ~300 rows (i.e. was written
			// with the full 5y history, not a stale compact/1y fetch).
			return prices, nil
		}
	}
	prices, err := h.FetchPrices(ticker)
	if err != nil {
		return nil, err
	}
	_ = cache.WritePrices(ticker, prices)
	return prices, nil
}

// fetchAVOverview returns the overview from cache if fresh, otherwise calls
// the Alpha Vantage OVERVIEW endpoint (free tier, 1 call/day per ticker).
func fetchAVOverview(h *api.Handler, ticker string) (indicators.Overview, error) {
	cachePath, err := cache.OverviewCachePath(ticker)
	if err == nil && cache.IsFresh(cachePath) {
		if ov, err := cache.ReadOverview(ticker); err == nil {
			return ov, nil
		}
	}
	ov, err := h.FetchOverview(ticker)
	if err != nil {
		return indicators.Overview{}, err
	}
	_ = cache.WriteOverview(ticker, ov)
	return ov, nil
}

// loadDotEnv reads KEY=VALUE pairs from a .env file and exports them as
// environment variables. Keys already present in the environment are not
// overwritten. Blank lines and lines starting with '#' are ignored.
func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}
