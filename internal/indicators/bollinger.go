package indicators

import "math"

// BollingerBands computes 20-period Bollinger Bands (±2σ) from closing prices.
//
// Signals:
//
//	Close < lower band  → BUY  (price oversold relative to recent range)
//	Close > upper band  → SELL (price overbought relative to recent range)
//	Otherwise           → HOLD
//
// %B = (close - lower) / (upper - lower); 0 = at lower band, 1 = at upper.
// Bandwidth = (upper - lower) / middle; low bandwidth indicates a squeeze.
func BollingerBands(prices []DailyPrice) (BollingerResult, ModelSignal) {
	const period = 20
	if len(prices) < period {
		return BollingerResult{}, SignalHold
	}

	window := prices[len(prices)-period:]
	sum := 0.0
	for _, p := range window {
		sum += p.Close
	}
	middle := sum / float64(period)

	variance := 0.0
	for _, p := range window {
		d := p.Close - middle
		variance += d * d
	}
	std := math.Sqrt(variance / float64(period))

	upper := middle + 2*std
	lower := middle - 2*std
	close := prices[len(prices)-1].Close

	bandwidth := 0.0
	if middle != 0 {
		bandwidth = (upper - lower) / middle
	}

	pctB := 0.0
	if upper != lower {
		pctB = (close - lower) / (upper - lower)
	}

	res := BollingerResult{
		Upper:     upper,
		Middle:    middle,
		Lower:     lower,
		PctB:      pctB,
		Bandwidth: bandwidth,
	}

	var sig ModelSignal
	switch {
	case close < lower:
		sig = SignalBuy
	case close > upper:
		sig = SignalSell
	default:
		sig = SignalHold
	}
	return res, sig
}
