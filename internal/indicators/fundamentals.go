package indicators

// EvaluateFundamentals scores PEG Ratio and Debt-to-Equity and returns a
// combined signal.
//
// PEG Ratio:
//
//	< 1.0   → BUY  (growth not yet priced in)
//	1.0–2.0 → HOLD (fairly valued relative to growth)
//	> 2.0   → SELL (growth premium is stretched)
//	0       → unscored (data unavailable)
//
// Debt-to-Equity:
//
//	< 0.5   → BUY  (conservatively financed)
//	0.5–1.5 → HOLD (manageable leverage)
//	> 1.5   → SELL (high leverage risk)
//	0       → unscored (data unavailable)
//
// Combined signal: average of available sub-signals (rounded). If neither
// metric is available the signal is HOLD.
func EvaluateFundamentals(ov Overview) FundamentalsResult {
	res := FundamentalsResult{
		PEGRatio:     ov.PEGRatio,
		DebtToEquity: ov.DebtToEquity,
		ROE:          ov.ROE,
	}

	scoredCount := 0
	scoreSum := 0

	if ov.PEGRatio > 0 {
		res.PEGSignal = scorePEG(ov.PEGRatio)
		scoreSum += int(res.PEGSignal)
		scoredCount++
	}

	if ov.DebtToEquity > 0 {
		res.DERatioSignal = scoreDE(ov.DebtToEquity)
		scoreSum += int(res.DERatioSignal)
		scoredCount++
	}

	if scoredCount == 0 {
		res.Combined = SignalHold
		return res
	}

	avg := float64(scoreSum) / float64(scoredCount)
	switch {
	case avg >= 0.5:
		res.Combined = SignalBuy
	case avg <= -0.5:
		res.Combined = SignalSell
	default:
		res.Combined = SignalHold
	}
	return res
}

func scorePEG(peg float64) ModelSignal {
	switch {
	case peg < 1.0:
		return SignalBuy
	case peg > 2.0:
		return SignalSell
	default:
		return SignalHold
	}
}

func scoreDE(de float64) ModelSignal {
	switch {
	case de < 0.5:
		return SignalBuy
	case de > 1.5:
		return SignalSell
	default:
		return SignalHold
	}
}
