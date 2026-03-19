// Package analysis provides technical-analysis indicators for stock prices.
package analysis

import (
	"errors"
	"math"
)

// ErrInsufficientData is returned when there are not enough data points to
// calculate an indicator.
var ErrInsufficientData = errors.New("insufficient data for indicator calculation")

// Result holds the computed technical indicators for a series of prices.
type Result struct {
	// SMA20 is the 20-period Simple Moving Average of the most recent close.
	SMA20 float64
	// SMA50 is the 50-period Simple Moving Average of the most recent close.
	SMA50 float64
	// RSI14 is the 14-period Relative Strength Index.
	RSI14 float64
	// MACD is the difference between the 12- and 26-period EMAs.
	MACD float64
	// MACDSignal is the 9-period EMA of MACD values.
	MACDSignal float64
	// MACDHistogram is MACD minus MACDSignal.
	MACDHistogram float64
	// LatestClose is the most recent closing price.
	LatestClose float64
}

// Analyze computes technical indicators from the provided closing-price series
// (oldest first). It requires at least 26 data points.
func Analyze(closes []float64) (Result, error) {
	if len(closes) < 26 {
		return Result{}, ErrInsufficientData
	}

	latest := closes[len(closes)-1]

	sma20, err := sma(closes, 20)
	if err != nil {
		return Result{}, err
	}

	sma50, _ := sma(closes, 50) // optional – may have insufficient data

	rsi14, err := rsi(closes, 14)
	if err != nil {
		return Result{}, err
	}

	macdVal, signal, histogram, err := macd(closes)
	if err != nil {
		return Result{}, err
	}

	return Result{
		SMA20:         sma20,
		SMA50:         sma50,
		RSI14:         rsi14,
		MACD:          macdVal,
		MACDSignal:    signal,
		MACDHistogram: histogram,
		LatestClose:   latest,
	}, nil
}

// sma computes the Simple Moving Average over the last `period` values.
func sma(data []float64, period int) (float64, error) {
	if len(data) < period {
		return 0, ErrInsufficientData
	}
	sum := 0.0
	for _, v := range data[len(data)-period:] {
		sum += v
	}
	return sum / float64(period), nil
}

// ema computes an Exponential Moving Average over the provided slice.
// It uses the standard smoothing factor k = 2/(period+1).
func ema(data []float64, period int) ([]float64, error) {
	if len(data) < period {
		return nil, ErrInsufficientData
	}

	k := 2.0 / float64(period+1)

	// Seed with a simple average of the first `period` values.
	seed := 0.0
	for _, v := range data[:period] {
		seed += v
	}
	seed /= float64(period)

	result := make([]float64, len(data)-period+1)
	result[0] = seed
	for i, v := range data[period:] {
		result[i+1] = v*k + result[i]*(1-k)
	}
	return result, nil
}

// rsi computes the Relative Strength Index using Wilder's smoothing method.
func rsi(data []float64, period int) (float64, error) {
	if len(data) < period+1 {
		return 0, ErrInsufficientData
	}

	// Calculate initial average gain / loss over the first `period` changes.
	var gainSum, lossSum float64
	for i := 1; i <= period; i++ {
		change := data[i] - data[i-1]
		if change > 0 {
			gainSum += change
		} else {
			lossSum += math.Abs(change)
		}
	}
	avgGain := gainSum / float64(period)
	avgLoss := lossSum / float64(period)

	// Wilder smoothing for remaining data points.
	for i := period + 1; i < len(data); i++ {
		change := data[i] - data[i-1]
		gain, loss := 0.0, 0.0
		if change > 0 {
			gain = change
		} else {
			loss = math.Abs(change)
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
	}

	if avgLoss == 0 {
		return 100, nil
	}
	rs := avgGain / avgLoss
	return 100 - 100/(1+rs), nil
}

// macd computes the MACD line, signal line, and histogram.
// MACD uses EMA(12) - EMA(26); signal uses EMA(9) of MACD values.
func macd(data []float64) (macdLine, signal, histogram float64, err error) {
	ema12, err := ema(data, 12)
	if err != nil {
		return 0, 0, 0, err
	}
	ema26, err := ema(data, 26)
	if err != nil {
		return 0, 0, 0, err
	}

	// Align the two EMA slices: ema26 is shorter by (26-12)=14 elements.
	// ema12 has len = len(data) - 12 + 1
	// ema26 has len = len(data) - 26 + 1
	// The MACD series spans the overlap.
	offset := len(ema12) - len(ema26)
	macdSeries := make([]float64, len(ema26))
	for i := range ema26 {
		macdSeries[i] = ema12[i+offset] - ema26[i]
	}

	if len(macdSeries) < 9 {
		return 0, 0, 0, ErrInsufficientData
	}

	signalSeries, err := ema(macdSeries, 9)
	if err != nil {
		return 0, 0, 0, err
	}

	macdLine = macdSeries[len(macdSeries)-1]
	signal = signalSeries[len(signalSeries)-1]
	histogram = macdLine - signal
	return macdLine, signal, histogram, nil
}
