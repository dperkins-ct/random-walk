package indicators

// OnBalanceVolume computes cumulative On-Balance Volume and its 20-day slope.
//
// OBV is incremented by the day's volume when close is up, decremented when
// close is down. The 20-day linear slope determines trend direction:
//
//	Slope > 0  → accumulation → BUY
//	Slope < 0  → distribution → SELL
//	Slope = 0  → neutral      → HOLD
func OnBalanceVolume(prices []DailyPrice) (OBVResult, ModelSignal) {
	if len(prices) < 2 {
		return OBVResult{}, SignalHold
	}

	// Build full OBV series.
	obvSeries := make([]float64, len(prices))
	obvSeries[0] = 0
	for i := 1; i < len(prices); i++ {
		diff := prices[i].Close - prices[i-1].Close
		switch {
		case diff > 0:
			obvSeries[i] = obvSeries[i-1] + float64(prices[i].Volume)
		case diff < 0:
			obvSeries[i] = obvSeries[i-1] - float64(prices[i].Volume)
		default:
			obvSeries[i] = obvSeries[i-1]
		}
	}

	currentOBV := int64(obvSeries[len(obvSeries)-1])

	// Average daily volume over the full window (used to normalize slope).
	volSum := 0.0
	for _, p := range prices {
		volSum += float64(p.Volume)
	}
	avgVol := volSum / float64(len(prices))

	// 20-day linear regression slope on the last 20 OBV values.
	const slopePeriod = 20
	slope := 0.0
	if len(obvSeries) >= slopePeriod {
		window := obvSeries[len(obvSeries)-slopePeriod:]
		slope = linearSlope(window)
	}

	res := OBVResult{OBV: currentOBV, Slope: slope, AvgDailyVolume: avgVol}

	var sig ModelSignal
	switch {
	case slope > 0:
		sig = SignalBuy
	case slope < 0:
		sig = SignalSell
	default:
		sig = SignalHold
	}
	return res, sig
}

// linearSlope returns the slope of the best-fit line through a series of values.
func linearSlope(vals []float64) float64 {
	n := float64(len(vals))
	if n < 2 {
		return 0
	}
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for i, v := range vals {
		x := float64(i)
		sumX += x
		sumY += v
		sumXY += x * v
		sumX2 += x * x
	}
	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0
	}
	return (n*sumXY - sumX*sumY) / denom
}
