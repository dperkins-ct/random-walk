package cache

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/dperkins-ct/random-walk/internal/indicators"
)

func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find home directory: %w", err)
	}
	dir := filepath.Join(home, ".random-walk", "cache")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create cache dir: %w", err)
	}
	return dir, nil
}

func pricesPath(ticker string) (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ticker+"_daily.csv"), nil
}

func overviewPath(ticker string) (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ticker+"_overview.csv"), nil
}

// IsFresh returns true if the file at path was written today (local time).
func IsFresh(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	now := time.Now()
	mod := info.ModTime()
	return mod.Year() == now.Year() && mod.YearDay() == now.YearDay()
}

// WritePrices persists a slice of DailyPrice to a CSV file.
func WritePrices(ticker string, prices []indicators.DailyPrice) error {
	path, err := pricesPath(ticker)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create cache file: %w", err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"date", "open", "high", "low", "close", "adj_close", "volume"})
	for _, p := range prices {
		_ = w.Write([]string{
			p.Date,
			strconv.FormatFloat(p.Open, 'f', 6, 64),
			strconv.FormatFloat(p.High, 'f', 6, 64),
			strconv.FormatFloat(p.Low, 'f', 6, 64),
			strconv.FormatFloat(p.Close, 'f', 6, 64),
			strconv.FormatFloat(p.AdjClose, 'f', 6, 64),
			strconv.FormatInt(p.Volume, 10),
		})
	}
	w.Flush()
	return w.Error()
}

// ReadPrices loads daily prices from the CSV cache.
func ReadPrices(ticker string) ([]indicators.DailyPrice, error) {
	path, err := pricesPath(ticker)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("cannot parse cache CSV: %w", err)
	}
	prices := make([]indicators.DailyPrice, 0, len(rows)-1)
	for _, row := range rows[1:] {
		if len(row) < 7 {
			continue
		}
		p := indicators.DailyPrice{Date: row[0]}
		p.Open, _ = strconv.ParseFloat(row[1], 64)
		p.High, _ = strconv.ParseFloat(row[2], 64)
		p.Low, _ = strconv.ParseFloat(row[3], 64)
		p.Close, _ = strconv.ParseFloat(row[4], 64)
		p.AdjClose, _ = strconv.ParseFloat(row[5], 64)
		p.Volume, _ = strconv.ParseInt(row[6], 10, 64)
		prices = append(prices, p)
	}
	return prices, nil
}

// PricesCachePath returns the cache file path for a ticker (for freshness checks).
func PricesCachePath(ticker string) (string, error) {
	return pricesPath(ticker)
}

// WriteOverview persists an Overview to a single-row CSV.
func WriteOverview(ticker string, ov indicators.Overview) error {
	path, err := overviewPath(ticker)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create overview cache: %w", err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"symbol", "name", "sector", "pe_ratio", "forward_pe", "peg_ratio", "price_to_book", "roe"})
	_ = w.Write([]string{
		ov.Symbol,
		ov.Name,
		ov.Sector,
		strconv.FormatFloat(ov.PERatio, 'f', 4, 64),
		strconv.FormatFloat(ov.ForwardPE, 'f', 4, 64),
		strconv.FormatFloat(ov.PEGRatio, 'f', 4, 64),
		strconv.FormatFloat(ov.PriceToBook, 'f', 4, 64),
		strconv.FormatFloat(ov.ROE, 'f', 4, 64),
	})
	w.Flush()
	return w.Error()
}

// ReadOverview loads an Overview from the CSV cache.
// Old cache files with fewer than 8 columns are treated as missing (return error)
// so the caller re-fetches and writes the extended format.
func ReadOverview(ticker string) (indicators.Overview, error) {
	path, err := overviewPath(ticker)
	if err != nil {
		return indicators.Overview{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return indicators.Overview{}, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return indicators.Overview{}, err
	}
	if len(rows) < 2 {
		return indicators.Overview{}, fmt.Errorf("empty overview cache")
	}
	// Require 8 columns; older 5-column caches trigger a re-fetch.
	row := rows[1]
	if len(row) < 8 {
		return indicators.Overview{}, fmt.Errorf("stale overview cache (missing extended columns)")
	}
	ov := indicators.Overview{Symbol: row[0], Name: row[1], Sector: row[2]}
	ov.PERatio, _ = strconv.ParseFloat(row[3], 64)
	ov.ForwardPE, _ = strconv.ParseFloat(row[4], 64)
	ov.PEGRatio, _ = strconv.ParseFloat(row[5], 64)
	ov.PriceToBook, _ = strconv.ParseFloat(row[6], 64)
	ov.ROE, _ = strconv.ParseFloat(row[7], 64)
	return ov, nil
}

// OverviewCachePath returns the cache file path for a ticker's overview.
func OverviewCachePath(ticker string) (string, error) {
	return overviewPath(ticker)
}
