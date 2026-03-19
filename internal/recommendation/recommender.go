// Package recommendation generates trading recommendations from analysis results.
package recommendation

import (
	"fmt"
	"strings"

	"github.com/dperkins-ct/random-walk/internal/analysis"
)

// Action represents the recommended trading action.
type Action string

const (
	// ActionBuy indicates a bullish signal.
	ActionBuy Action = "BUY"
	// ActionSell indicates a bearish signal.
	ActionSell Action = "SELL"
	// ActionHold indicates a neutral signal.
	ActionHold Action = "HOLD"
)

// Recommendation bundles the overall action with the individual signal scores.
type Recommendation struct {
	Ticker  string
	Action  Action
	Score   int    // net bullish signals minus bearish signals
	Reasons []string
}

// Generate evaluates the technical analysis result and produces a Recommendation.
func Generate(ticker string, r analysis.Result) Recommendation {
	var bullish, bearish []string

	// --- RSI signal ---
	switch {
	case r.RSI14 < 30:
		bullish = append(bullish, fmt.Sprintf("RSI %.1f is oversold (<30) → bullish reversal candidate", r.RSI14))
	case r.RSI14 > 70:
		bearish = append(bearish, fmt.Sprintf("RSI %.1f is overbought (>70) → bearish reversal candidate", r.RSI14))
	default:
		bullish = append(bullish, fmt.Sprintf("RSI %.1f is neutral (30–70)", r.RSI14))
	}

	// --- SMA20 / price signal ---
	if r.LatestClose > r.SMA20 {
		bullish = append(bullish, fmt.Sprintf("Close $%.2f is above SMA20 $%.2f → short-term uptrend", r.LatestClose, r.SMA20))
	} else {
		bearish = append(bearish, fmt.Sprintf("Close $%.2f is below SMA20 $%.2f → short-term downtrend", r.LatestClose, r.SMA20))
	}

	// --- SMA50 / price signal (only when available) ---
	if r.SMA50 > 0 {
		if r.LatestClose > r.SMA50 {
			bullish = append(bullish, fmt.Sprintf("Close $%.2f is above SMA50 $%.2f → medium-term uptrend", r.LatestClose, r.SMA50))
		} else {
			bearish = append(bearish, fmt.Sprintf("Close $%.2f is below SMA50 $%.2f → medium-term downtrend", r.LatestClose, r.SMA50))
		}
	}

	// --- MACD signal ---
	if r.MACDHistogram > 0 {
		bullish = append(bullish, fmt.Sprintf("MACD histogram $%.4f > 0 → bullish momentum", r.MACDHistogram))
	} else {
		bearish = append(bearish, fmt.Sprintf("MACD histogram $%.4f < 0 → bearish momentum", r.MACDHistogram))
	}

	// --- Determine overall action ---
	score := len(bullish) - len(bearish)
	var action Action
	switch {
	case score > 0:
		action = ActionBuy
	case score < 0:
		action = ActionSell
	default:
		action = ActionHold
	}

	// Combine reasons in a readable order (bullish first, then bearish).
	reasons := make([]string, 0, len(bullish)+len(bearish))
	reasons = append(reasons, bullish...)
	reasons = append(reasons, bearish...)

	return Recommendation{
		Ticker:  strings.ToUpper(ticker),
		Action:  action,
		Score:   score,
		Reasons: reasons,
	}
}
