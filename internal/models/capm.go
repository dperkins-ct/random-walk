package models

// alignedReturns aligns two price series by date and returns paired daily returns.
// Both slices must be chronological (oldest first).
func alignedReturns(stock, market []DailyPrice) ([]float64, []float64) {
	mktMap := make(map[string]float64, len(market))
	for i := 1; i < len(market); i++ {
		r := (market[i].AdjClose - market[i-1].AdjClose) / market[i-1].AdjClose
		mktMap[market[i].Date] = r
	}
	var sReturns, mReturns []float64
	for i := 1; i < len(stock); i++ {
		date := stock[i].Date
		if mktReturn, ok := mktMap[date]; ok {
			sReturn := (stock[i].AdjClose - stock[i-1].AdjClose) / stock[i-1].AdjClose
			sReturns = append(sReturns, sReturn)
			mReturns = append(mReturns, mktReturn)
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

// CAPM computes Beta, Expected Return (via CAPM), and Jensen's Alpha.
// stockPrices and marketPrices must be chronological (oldest first).
// annualRiskFreeRate and annualMarketReturn should be decimal (e.g. 0.043).
func CAPM(stockPrices, marketPrices []DailyPrice, annualRiskFreeRate, annualMarketReturn float64) CAPMResult {
	sReturns, mReturns := alignedReturns(stockPrices, marketPrices)
	if len(sReturns) < 2 {
		return CAPMResult{}
	}
	beta := covariance(sReturns, mReturns) / variance(mReturns)
	expectedReturn := annualRiskFreeRate + beta*(annualMarketReturn-annualRiskFreeRate)
	actualAnnualReturn := mean(sReturns) * 252
	alpha := actualAnnualReturn - expectedReturn
	return CAPMResult{
		Beta:           beta,
		ExpectedReturn: expectedReturn,
		Alpha:          alpha,
	}
}
