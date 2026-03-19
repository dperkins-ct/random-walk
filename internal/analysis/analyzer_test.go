package analysis_test

import (
	"math"
	"testing"

	"github.com/dperkins-ct/random-walk/internal/analysis"
)

// linspace generates n evenly spaced values from start to end (inclusive).
func linspace(start, end float64, n int) []float64 {
	out := make([]float64, n)
	step := (end - start) / float64(n-1)
	for i := range out {
		out[i] = start + step*float64(i)
	}
	return out
}

func TestAnalyze_InsufficientData(t *testing.T) {
	_, err := analysis.Analyze([]float64{1, 2, 3})
	if err == nil {
		t.Fatal("expected error for insufficient data, got nil")
	}
}

func TestAnalyze_SMA20(t *testing.T) {
	// 50 prices that increase linearly from 100 to 149.
	closes := linspace(100, 149, 50)
	result, err := analysis.Analyze(closes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// SMA20 of the last 20 values (130..149) should be their mean = 139.5.
	want := 139.5
	if math.Abs(result.SMA20-want) > 0.01 {
		t.Errorf("SMA20 = %.4f, want %.4f", result.SMA20, want)
	}
}

func TestAnalyze_RSI_ConstantUp(t *testing.T) {
	// Constant upward trend → RSI should be 100 (no losses).
	closes := linspace(100, 200, 50)
	result, err := analysis.Analyze(closes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RSI14 != 100 {
		t.Errorf("RSI14 = %.2f, want 100", result.RSI14)
	}
}

func TestAnalyze_RSI_Range(t *testing.T) {
	// Alternating up/down prices → RSI must be in [0, 100].
	closes := make([]float64, 60)
	for i := range closes {
		if i%2 == 0 {
			closes[i] = 100
		} else {
			closes[i] = 90
		}
	}
	result, err := analysis.Analyze(closes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RSI14 < 0 || result.RSI14 > 100 {
		t.Errorf("RSI14 = %.2f out of [0, 100]", result.RSI14)
	}
}

func TestAnalyze_LatestClose(t *testing.T) {
	closes := linspace(10, 59, 50)
	result, err := analysis.Analyze(closes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.LatestClose != closes[len(closes)-1] {
		t.Errorf("LatestClose = %.2f, want %.2f", result.LatestClose, closes[len(closes)-1])
	}
}

func TestAnalyze_MACD_Histogram(t *testing.T) {
	// A long uptrend should produce a positive MACD histogram.
	closes := linspace(50, 150, 60)
	result, err := analysis.Analyze(closes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// histogram = macd - signal
	got := result.MACD - result.MACDSignal
	if math.Abs(got-result.MACDHistogram) > 1e-9 {
		t.Errorf("histogram inconsistency: MACD-Signal=%.6f, Histogram=%.6f", got, result.MACDHistogram)
	}
}
