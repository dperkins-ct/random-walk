package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/dperkins-ct/random-walk/internal/analysis"
	"github.com/dperkins-ct/random-walk/internal/api"
	"github.com/dperkins-ct/random-walk/internal/cache"
	"github.com/dperkins-ct/random-walk/internal/models"
	"github.com/dperkins-ct/random-walk/internal/output"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("random-walk", flag.ContinueOnError)
	apiKey := fs.String("api-key", "", "Alpha Vantage API key (or set ALPHAVANTAGE_API_KEY)")
	period := fs.String("period", "1y", "Historical lookback: 1y | 2y | 5y")
	riskFreeRate := fs.Float64("risk-free-rate", 0.043, "Annual risk-free rate as decimal (e.g. 0.043)")
	marketReturn := fs.Float64("market-return", 0.10, "Expected annual market return as decimal (e.g. 0.10)")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		fs.Usage()
		return errors.New("ticker symbol required  e.g.:  random-walk AAPL --api-key YOUR_KEY")
	}
	ticker := fs.Arg(0)

	key := *apiKey
	if key == "" {
		key = os.Getenv("ALPHAVANTAGE_API_KEY")
	}
	if key == "" {
		return errors.New("API key required: use --api-key or set ALPHAVANTAGE_API_KEY env var")
	}

	// For periods > 1 year we need the full history from Alpha Vantage.
	outputSize := "compact"
	if *period == "2y" || *period == "5y" {
		outputSize = "full"
	}

	apiHandler := api.NewHandler(key)
	analysisHandler := analysis.NewHandler(*riskFreeRate, *marketReturn)

	// Fetch stock prices (cache-aware).
	stockPrices, err := fetchPrices(apiHandler, ticker, outputSize)
	if err != nil {
		return fmt.Errorf("stock data: %w", err)
	}

	// Fetch SPY prices for CAPM market benchmark (cache-aware).
	spyPrices, err := fetchPrices(apiHandler, "SPY", outputSize)
	if err != nil {
		return fmt.Errorf("market benchmark (SPY): %w", err)
	}

	// Fetch fundamental overview (cache-aware, non-fatal on failure).
	overview, err := fetchOverview(apiHandler, ticker)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: overview unavailable for %s: %v\n", ticker, err)
	}

	// Run all analysis models.
	result, err := analysisHandler.Analyze(ticker, stockPrices, spyPrices, overview, *period)
	if err != nil {
		return fmt.Errorf("analysis: %w", err)
	}

	// Render colored terminal report.
	output.Print(result)
	return nil
}

// fetchPrices returns prices from cache if fresh, otherwise calls the API.
func fetchPrices(h *api.Handler, ticker, outputSize string) ([]models.DailyPrice, error) {
	cachePath, err := cache.PricesCachePath(ticker)
	if err == nil && cache.IsFresh(cachePath) {
		if prices, err := cache.ReadPrices(ticker); err == nil {
			return prices, nil
		}
	}
	prices, err := h.FetchPrices(ticker, outputSize)
	if err != nil {
		return nil, err
	}
	_ = cache.WritePrices(ticker, prices)
	return prices, nil
}

// fetchOverview returns the overview from cache if fresh, otherwise calls the API.
func fetchOverview(h *api.Handler, ticker string) (models.Overview, error) {
	cachePath, err := cache.OverviewCachePath(ticker)
	if err == nil && cache.IsFresh(cachePath) {
		if ov, err := cache.ReadOverview(ticker); err == nil {
			return ov, nil
		}
	}
	ov, err := h.FetchOverview(ticker)
	if err != nil {
		return models.Overview{}, err
	}
	_ = cache.WriteOverview(ticker, ov)
	return ov, nil
}
