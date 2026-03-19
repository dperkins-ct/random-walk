package indicators

// RSI computes the 14-period Relative Strength Index using Wilder's smoothing.
//
// Interpretation:
//   RSI < 30  -> oversold  -> BUY signal
//   RSI > 70  -> overbought -> SELL signal
//   30-70     -> neutral
//
// Returns 50 (neutral) if there are fewer than 15 data points.
func RSI(prices []DailyPrice) float64 {
	const period = 14
	if len(prices) < period+1 {
		return 50
	}
	returns := dailyReturns(prices)

	// Seed: average of first `period` gains and losses.
	initGain, initLoss := 0.0, 0.0
	for _, r := range returns[:period] {
		if r > 0 {
			initGain += r
		} else {
			initLoss += -r
		}
	}
	avgGain := initGain / float64(period)
	avgLoss := initLoss / float64(period)

	// Wilder's smoothing for subsequent values.
	for _, r := range returns[period:] {
		gain, loss := 0.0, 0.0
		if r > 0 {
			gain = r
		} else {
			loss = -r
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
	}

	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// RSISignal converts an RSI value to a ModelSignal.
func RSISignal(rsi float64) ModelSignal {
	switch {
	case rsi < 30:
		return SignalBuy  // oversold
	case rsi > 70:
		return SignalSell // overbought
	default:
		return SignalHold
	}
}
