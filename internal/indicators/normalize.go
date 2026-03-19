package indicators

// normalize.go maps each raw indicator value to a continuous score in [-1, +1]
// using domain-calibrated linear transforms anchored to known reference points.
//
// Each function is designed so that:
//   -1.0 = strongly bearish / high risk
//    0.0 = neutral
//   +1.0 = strongly bullish / low risk
//
// Values beyond the anchor points are clamped to ±1 so that extreme outliers
// can't dominate the composite score.
//
// These normalized values are multiplied by the same weights used previously
// (wHigh=2.0, wMid=1.5, wStd=1.0, wLow=0.5), giving a composite in [-15.5, +15.5].

// clamp constrains v to [lo, hi].
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// NormalizeSharpe maps Sharpe Ratio to [-1, +1].
//
//	Anchors: 0.0 → -1  |  0.5 → 0  |  1.5 → +1
//	Formula: clamp((v - 0.5) / 1.0)
func NormalizeSharpe(v float64) float64 {
	return clamp((v-0.5)/1.0, -1, 1)
}

// NormalizeSortino maps Sortino Ratio to [-1, +1].
//
//	Anchors: 0.0 → -1  |  0.75 → 0  |  2.0 → +1
//	Formula: clamp((v - 0.75) / 1.25)
func NormalizeSortino(v float64) float64 {
	return clamp((v-0.75)/1.25, -1, 1)
}

// NormalizeAlpha maps Jensen's Alpha (decimal, e.g. 0.05 = 5%) to [-1, +1].
//
//	Anchors: -5% → -1  |  0% → 0  |  +5% → +1
//	Formula: clamp(alpha / 0.05)
func NormalizeAlpha(alpha float64) float64 {
	return clamp(alpha/0.05, -1, 1)
}

// NormalizeMA maps the three Moving Average conditions to a continuous [-1, +1]
// score by counting how many of the three sub-conditions are bullish.
//
// Sub-conditions (each contributes 2/3 of a point toward +1):
//
//	EMA12 > EMA26, SMA20 > SMA50, MACD > Signal
//
// Score = (bullishCount×2 - 3) / 3  → maps [0 bullish, 3 bullish] to [-1, +1].
// Partial agreement (1 or 2 of 3) gives intermediate values ≈ -0.33 and +0.33.
func NormalizeMA(ma MAResult) float64 {
	bullish := 0
	if ma.EMA12 > ma.EMA26 {
		bullish++
	}
	if ma.SMA20 > ma.SMA50 {
		bullish++
	}
	if ma.MACD > ma.Signal {
		bullish++
	}
	return (float64(bullish)*2 - 3) / 3
}

// NormalizeRSI maps RSI (0–100) to [-1, +1].
// RSI is a contrarian indicator — oversold is bullish, overbought is bearish.
//
//	Anchors: 70 → -1  |  50 → 0  |  30 → +1
//	Formula: clamp((50 - RSI) / 20)
func NormalizeRSI(rsi float64) float64 {
	return clamp((50-rsi)/20, -1, 1)
}

// NormalizePE maps P/E ratio to [-1, +1].
// Returns 0 if P/E is unavailable (≤ 0).
//
//	Anchors: 40 → -1  |  25 → 0  |  10 → +1
//	Formula: clamp((25 - PE) / 15)
func NormalizePE(pe float64) float64 {
	if pe <= 0 {
		return 0
	}
	return clamp((25-pe)/15, -1, 1)
}

// NormalizeBollingerPctB maps Bollinger %B to [-1, +1].
// %B is contrarian — below the lower band is bullish, above the upper is bearish.
//
//	Anchors: 1.0 → -1  |  0.5 → 0  |  0.0 → +1
//	Formula: clamp((0.5 - pctB) / 0.5)
func NormalizeBollingerPctB(pctB float64) float64 {
	return clamp((0.5-pctB)/0.5, -1, 1)
}

// NormalizeOBVSlope maps the OBV 20-day slope to [-1, +1] relative to average
// daily volume. A slope of +10% of avg daily volume per day → +1; -10% → -1.
//
// Normalizing by volume makes the score comparable across stocks with vastly
// different share volumes (e.g. NVDA vs BRK.A).
//
// Returns 0 if avgDailyVolume is zero (degenerate input).
func NormalizeOBVSlope(slope, avgDailyVolume float64) float64 {
	if avgDailyVolume == 0 {
		return 0
	}
	// 10% of avg daily volume per day is the ±1 anchor.
	anchor := avgDailyVolume * 0.10
	return clamp(slope/anchor, -1, 1)
}

// NormalizeRS maps the Relative Strength ratio (stock/SPY) to [-1, +1].
//
//	Anchors: 0.90 → -1  |  1.0 → 0  |  1.10 → +1
//	Formula: clamp((RS - 1.0) / 0.10)
func NormalizeRS(rs float64) float64 {
	return clamp((rs-1.0)/0.10, -1, 1)
}

// NormalizeMaxDrawdown maps peak-to-trough drawdown (positive decimal) to [-1, +1].
// Lower drawdown is better (bullish); higher drawdown is bearish.
//
//	Anchors: 0.30 → -1  |  0.15 → 0  |  0.0 → +1
//	Formula: clamp((0.15 - DD) / 0.15)
func NormalizeMaxDrawdown(dd float64) float64 {
	return clamp((0.15-dd)/0.15, -1, 1)
}

// NormalizeVaR maps the 95% daily VaR (negative decimal, e.g. -0.025 = -2.5%)
// to [-1, +1]. Less negative (smaller daily loss) is better.
//
//	Anchors: -0.03 → -1  |  -0.02 → 0  |  -0.01 → +1
//	Formula: clamp((var95 + 0.02) / 0.01)
func NormalizeVaR(var95 float64) float64 {
	return clamp((var95+0.02)/0.01, -1, 1)
}

// NormalizeFundamentals maps PEG and P/B to a combined [-1, +1] score.
// Only available (> 0) metrics are included; returns 0 if both are unavailable.
//
// PEG anchors: 2.0 → -1  |  1.0 → 0  |  0.0 → +1   formula: clamp((1 - PEG) / 1.0)
// P/B  anchors: 4.0 → -1  |  2.0 → 0  |  0.5 → +1   formula: clamp((2 - PB) / 1.5)
//
// Combined = average of available sub-scores.
func NormalizeFundamentals(peg, pb float64) float64 {
	sum := 0.0
	count := 0

	if peg > 0 {
		sum += clamp((1.0-peg)/1.0, -1, 1)
		count++
	}
	if pb > 0 {
		sum += clamp((2.0-pb)/1.5, -1, 1)
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}
