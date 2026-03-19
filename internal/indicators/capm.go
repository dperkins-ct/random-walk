package indicators

// alignedReturns aligns two price series by date and returns paired daily returns.
// Both slices must be chronological (oldest first).
func alignedReturns(stock, market []DailyPrice) ([]float64, []float64) {
	mktMap := make(map[string]float64, len(market))
	for i := 1; i < len(market); i++ {
		if market[i-1].AdjClose > 0 {
			mktMap[market[i].Date] = (market[i].AdjClose - market[i-1].AdjClose) / market[i-1].AdjClose
		}
	}
	var sReturns, mReturns []float64
	for i := 1; i < len(stock); i++ {
		date := stock[i].Date
		if mktReturn, ok := mktMap[date]; ok {
			if stock[i-1].AdjClose > 0 {
				sReturn := (stock[i].AdjClose - stock[i-1].AdjClose) / stock[i-1].AdjClose
				sReturns = append(sReturns, sReturn)
				mReturns = append(mReturns, mktReturn)
			}
		}
	}
	return sReturns, mReturns
}

// covariance computes the population covariance of two equally-sized slices.
func covariance(xs, ys []float64) float64 {
	if len(xs) != len(ys) || len(xs) == 0 {
		return 0
	}
	mx := mean(xs)
	my := mean(ys)
	cov := 0.0
	for i := range xs {
		cov += (xs[i] - mx) * (ys[i] - my)
	}
	return cov / float64(len(xs))
}

// variance computes the population variance.
func variance(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	m := mean(xs)
	v := 0.0
	for _, x := range xs {
		d := x - m
		v += d * d
	}
	return v / float64(len(xs))
}

// CAPM computes Beta, Expected Return, and Jensen's Alpha for a stock.
// It uses the actual realised market return from the period
// (mean(mktDailyReturns) x 252) rather than a long-run constant, so Alpha
// answers the question: "did this stock beat the market on a risk-adjusted
// basis over this specific period?"
func CAPM(stockPrices, marketPrices []DailyPrice, annualRiskFreeRate float64) CAPMResult {
	sReturns, mReturns := alignedReturns(stockPrices, marketPrices)
	if len(sReturns) < 2 {
		return CAPMResult{}
	}

	beta := covariance(sReturns, mReturns) / variance(mReturns)

	actualMktReturn := mean(mReturns) * 252
	expectedReturn := annualRiskFreeRate + beta*(actualMktReturn-annualRiskFreeRate)
	actualStockReturn := mean(sReturns) * 252
	alpha := actualStockReturn - expectedReturn

	return CAPMResult{
		Beta:               beta,
		ActualMarketReturn: actualMktReturn,
		ExpectedReturn:     expectedReturn,
		Alpha:              alpha,
	}
}
