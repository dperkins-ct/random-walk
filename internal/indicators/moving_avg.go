package indicators

// ema computes the Exponential Moving Average series for a given period.
// prices must be chronological (oldest first).
func ema(prices []float64, period int) []float64 {
	if len(prices) < period {
		return nil
	}
	k := 2.0 / float64(period+1)
	result := make([]float64, len(prices))

	// Seed with SMA of first `period` values.
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	result[period-1] = sum / float64(period)

	for i := period; i < len(prices); i++ {
		result[i] = prices[i]*k + result[i-1]*(1-k)
	}
	return result
}

// sma computes the Simple Moving Average over the last `period` values.
func sma(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

// MovingAverages computes SMA20, SMA50, EMA12, EMA26, MACD, and a signal line.
// prices must be chronological (oldest first) and have at least 50 data points.
func MovingAverages(prices []DailyPrice) MAResult {
	if len(prices) < 50 {
		return MAResult{Trend: Neutral}
	}
	closes := make([]float64, len(prices))
	for i, p := range prices {
		closes[i] = p.AdjClose
	}

	sma20val := sma(closes, 20)
	sma50val := sma(closes, 50)
	ema12vals := ema(closes, 12)
	ema26vals := ema(closes, 26)

	if ema12vals == nil || ema26vals == nil {
		return MAResult{SMA20: sma20val, SMA50: sma50val, Trend: Neutral}
	}

	// MACD line = EMA12 - EMA26 for each valid position (index >= 25).
	macdLine := make([]float64, len(closes))
	for i := 25; i < len(closes); i++ {
		macdLine[i] = ema12vals[i] - ema26vals[i]
	}

	// Signal line = 9-period EMA of the MACD values (from index 25 onward).
	macdSlice := macdLine[25:]
	signalVals := ema(macdSlice, 9)

	lastEMA12 := ema12vals[len(ema12vals)-1]
	lastEMA26 := ema26vals[len(ema26vals)-1]
	lastMACD := macdLine[len(macdLine)-1]

	var lastSignal float64
	if signalVals != nil {
		lastSignal = signalVals[len(signalVals)-1]
	}

	bullish := 0
	bearish := 0
	if lastEMA12 > lastEMA26 {
		bullish++
	} else {
		bearish++
	}
	if sma20val > sma50val {
		bullish++
	} else {
		bearish++
	}
	if lastMACD > lastSignal {
		bullish++
	} else {
		bearish++
	}

	trend := Neutral
	if bullish >= 2 {
		trend = Bullish
	} else if bearish >= 2 {
		trend = Bearish
	}

	return MAResult{
		SMA20:  sma20val,
		SMA50:  sma50val,
		EMA12:  lastEMA12,
		EMA26:  lastEMA26,
		MACD:   lastMACD,
		Signal: lastSignal,
		Trend:  trend,
	}
}
