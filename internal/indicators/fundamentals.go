package indicators

// EvaluateFundamentals scores PEG Ratio and Price-to-Book and returns a
// combined signal.
//
// PEG Ratio:
//
//	< 1.0     → BUY  (growth not yet priced in)
//	1.0–2.0   → HOLD (fairly valued relative to growth)
//	> 2.0     → SELL (growth premium is stretched)
//
// Price-to-Book (P/B):
//
//	< 1.0     → BUY  (trading below book value — potentially very cheap)
//	1.0–4.0   → HOLD (normal range across most sectors)
//	> 4.0     → SELL (high premium to assets; priced for perfection)
//
// Combined signal: average of available sub-signals (rounded).
// If neither metric is available the signal is HOLD.
func EvaluateFundamentals(ov Overview) FundamentalsResult {
	res := FundamentalsResult{
		PEGRatio:    ov.PEGRatio,
		PriceToBook: ov.PriceToBook,
		ROE:         ov.ROE,
	}

	scoredCount := 0
	scoreSum := 0

	if ov.PEGRatio > 0 {
		res.PEGSignal = scorePEG(ov.PEGRatio)
		scoreSum += int(res.PEGSignal)
		scoredCount++
	}

	if ov.PriceToBook > 0 {
		res.PBSignal = scorePB(ov.PriceToBook)
		scoreSum += int(res.PBSignal)
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

func scorePB(pb float64) ModelSignal {
	switch {
	case pb < 1.0:
		return SignalBuy
	case pb > 4.0:
		return SignalSell
	default:
		return SignalHold
	}
}
