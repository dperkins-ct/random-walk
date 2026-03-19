package indicators

// RelativeStrength computes how a stock performed relative to SPY (and
// optionally a sector ETF) over the analysis period.
//
// RS = (1 + stock cumulative return) / (1 + benchmark cumulative return)
//
//	RS > 1.10  → outperforming  → BUY
//	RS < 0.90  → underperforming → SELL
//	Otherwise  → HOLD
//
// Both slices must be chronological (oldest first) and already filtered to the
// desired analysis period. Alignment is date-based; unmatched dates are skipped.
func RelativeStrength(stock, spy []DailyPrice, sectorPrices []DailyPrice) (RSResult, ModelSignal) {
	rsSPY := cumulativeReturnRatio(stock, spy)

	rsSector := 0.0
	if len(sectorPrices) >= 2 {
		rsSector = cumulativeReturnRatio(stock, sectorPrices)
	}

	res := RSResult{VsSPY: rsSPY, VsSector: rsSector}

	var sig ModelSignal
	switch {
	case rsSPY > 1.10:
		sig = SignalBuy
	case rsSPY < 0.90:
		sig = SignalSell
	default:
		sig = SignalHold
	}
	return res, sig
}

// cumulativeReturnRatio computes (1+stockReturn)/(1+benchReturn) over the
// overlapping date range of both slices.
func cumulativeReturnRatio(stock, bench []DailyPrice) float64 {
	// Build date→close maps.
	benchMap := make(map[string]float64, len(bench))
	for _, p := range bench {
		benchMap[p.Date] = p.Close
	}

	// Find the first overlapping date.
	startStockClose := 0.0
	startBenchClose := 0.0
	endStockClose := 0.0
	endBenchClose := 0.0
	found := false

	for _, p := range stock {
		bc, ok := benchMap[p.Date]
		if !ok {
			continue
		}
		if !found {
			startStockClose = p.Close
			startBenchClose = bc
			found = true
		}
		endStockClose = p.Close
		endBenchClose = bc
	}

	if !found || startStockClose == 0 || startBenchClose == 0 {
		return 1.0 // neutral if no overlap
	}

	stockReturn := (endStockClose - startStockClose) / startStockClose
	benchReturn := (endBenchClose - startBenchClose) / startBenchClose

	if 1+benchReturn == 0 {
		return 1.0
	}
	return (1 + stockReturn) / (1 + benchReturn)
}
