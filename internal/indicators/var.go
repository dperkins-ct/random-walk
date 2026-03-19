package indicators

import "sort"

// ValueAtRisk computes the 95% historical VaR and Conditional VaR (CVaR/ES)
// from daily log returns.
//
// VaR95 is the 5th-percentile daily return — on the worst 5% of days the loss
// is expected to equal or exceed this magnitude.
// CVaR is the mean of returns at or below VaR95 (expected shortfall).
//
// Signals:
//
//	VaR95 < -3%   → SELL  (tail risk is high)
//	VaR95 < -2%   → HOLD  (moderate tail risk)
//	VaR95 ≥ -2%   → BUY   (contained tail risk)
func ValueAtRisk(prices []DailyPrice) (VaRResult, ModelSignal) {
	if len(prices) < 20 {
		return VaRResult{}, SignalHold
	}

	returns := make([]float64, 0, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if prices[i-1].Close > 0 {
			r := (prices[i].Close - prices[i-1].Close) / prices[i-1].Close
			returns = append(returns, r)
		}
	}
	if len(returns) == 0 {
		return VaRResult{}, SignalHold
	}

	sorted := make([]float64, len(returns))
	copy(sorted, returns)
	sort.Float64s(sorted)

	// 5th percentile index (round down).
	idx := int(float64(len(sorted)) * 0.05)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	var95 := sorted[idx]

	// CVaR: mean of all returns at or below VaR95.
	tailSum := 0.0
	tailCount := 0
	for _, r := range sorted {
		if r <= var95 {
			tailSum += r
			tailCount++
		}
	}
	cvar := 0.0
	if tailCount > 0 {
		cvar = tailSum / float64(tailCount)
	}

	res := VaRResult{VaR95: var95, CVaR: cvar}

	var sig ModelSignal
	switch {
	case var95 < -0.03:
		sig = SignalSell
	case var95 < -0.02:
		sig = SignalHold
	default:
		sig = SignalBuy
	}
	return res, sig
}
