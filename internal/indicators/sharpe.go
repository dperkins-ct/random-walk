package indicators

import "math"

// RiskFreeRateDaily converts an annual risk-free rate to a daily rate.
func RiskFreeRateDaily(annualRate float64) float64 {
	return annualRate / 252.0
}

// dailyReturns computes percentage daily returns from a slice of DailyPrice.
// Prices must be in chronological order (oldest first).
func dailyReturns(prices []DailyPrice) []float64 {
	if len(prices) < 2 {
		return nil
	}
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = (prices[i].AdjClose - prices[i-1].AdjClose) / prices[i-1].AdjClose
	}
	return returns
}

// mean returns the arithmetic mean of a slice.
func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

// stddev returns the population standard deviation of a slice.
func stddev(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	m := mean(xs)
	variance := 0.0
	for _, x := range xs {
		diff := x - m
		variance += diff * diff
	}
	return math.Sqrt(variance / float64(len(xs)))
}

// SharpeRatio computes the annualized Sharpe Ratio.
// prices must be chronological (oldest first).
func SharpeRatio(prices []DailyPrice, annualRiskFreeRate float64) float64 {
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
	sd := stddev(excessReturns)
	if sd == 0 {
		return 0
	}
	return (avg / sd) * math.Sqrt(252)
}
