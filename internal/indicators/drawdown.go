package indicators

import "math"

// MaxDrawdown computes the worst peak-to-trough decline in closing prices and
// the Calmar Ratio (annualized return / |max drawdown|).
//
// Signals:
//
//	MaxDrawdown > 40%  → SELL (severe historical loss)
//	MaxDrawdown > 25%  → HOLD (material drawdown, caution warranted)
//	MaxDrawdown ≤ 25%  → BUY  (contained downside)
func MaxDrawdown(prices []DailyPrice) (DrawdownResult, ModelSignal) {
	if len(prices) < 2 {
		return DrawdownResult{}, SignalHold
	}

	peak := prices[0].Close
	maxDD := 0.0
	for _, p := range prices[1:] {
		if p.Close > peak {
			peak = p.Close
		}
		if peak > 0 {
			dd := (peak - p.Close) / peak
			if dd > maxDD {
				maxDD = dd
			}
		}
	}

	// Annualized return over the period (simple).
	startClose := prices[0].Close
	endClose := prices[len(prices)-1].Close
	tradingDays := float64(len(prices))
	annualReturn := 0.0
	if startClose > 0 && tradingDays > 0 {
		totalReturn := (endClose - startClose) / startClose
		annualReturn = math.Pow(1+totalReturn, 252/tradingDays) - 1
	}

	calmar := 0.0
	if maxDD > 0 {
		calmar = annualReturn / maxDD
	}

	res := DrawdownResult{MaxDrawdown: maxDD, Calmar: calmar}

	var sig ModelSignal
	switch {
	case maxDD > 0.40:
		sig = SignalSell
	case maxDD > 0.25:
		sig = SignalHold
	default:
		sig = SignalBuy
	}
	return res, sig
}
