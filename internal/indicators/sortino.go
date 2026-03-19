package indicators

import "math"

// SortinoRatio computes the annualized Sortino Ratio.
// Only downside volatility (returns below zero) is penalized.
// prices must be chronological (oldest first).
func SortinoRatio(prices []DailyPrice, annualRiskFreeRate float64) float64 {
	returns := dailyReturns(prices)
	if len(returns) == 0 {
		return 0
	}
	rfDaily := RiskFreeRateDaily(annualRiskFreeRate)
	excessReturns := make([]float64, len(returns))
	for i, r := range returns {
		excessReturns[i] = r - rfDaily
	}
	avg := mean(excessReturns)

	// Downside deviation: only returns below 0 contribute.
	downsideVariance := 0.0
	for _, r := range excessReturns {
		if r < 0 {
			downsideVariance += r * r
		}
	}
	downsideStddev := math.Sqrt(downsideVariance / float64(len(excessReturns)))
	if downsideStddev == 0 {
		return 0
	}
	return (avg / downsideStddev) * math.Sqrt(252)
}
