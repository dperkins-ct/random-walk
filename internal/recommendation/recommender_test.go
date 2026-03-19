package recommendation_test

import (
	"testing"

	"github.com/dperkins-ct/random-walk/internal/analysis"
	"github.com/dperkins-ct/random-walk/internal/recommendation"
)

func TestGenerate_BuySignal(t *testing.T) {
	// Strongly bullish indicators should yield BUY.
	r := analysis.Result{
		SMA20:         90,
		SMA50:         85,
		RSI14:         45, // neutral RSI → bullish
		MACD:          0.5,
		MACDSignal:    0.3,
		MACDHistogram: 0.2, // positive → bullish
		LatestClose:   100, // above both SMAs → bullish x2
	}
	rec := recommendation.Generate("AAPL", r)
	if rec.Action != recommendation.ActionBuy {
		t.Errorf("expected BUY, got %s (score %d)", rec.Action, rec.Score)
	}
}

func TestGenerate_SellSignal(t *testing.T) {
	// Strongly bearish indicators should yield SELL.
	r := analysis.Result{
		SMA20:         110,
		SMA50:         105,
		RSI14:         80, // overbought → bearish
		MACD:          -0.5,
		MACDSignal:    -0.2,
		MACDHistogram: -0.3, // negative → bearish
		LatestClose:   100,  // below both SMAs → bearish x2
	}
	rec := recommendation.Generate("MSFT", r)
	if rec.Action != recommendation.ActionSell {
		t.Errorf("expected SELL, got %s (score %d)", rec.Action, rec.Score)
	}
}

func TestGenerate_HoldSignal(t *testing.T) {
	// Mixed signals (equal bullish & bearish) should yield HOLD.
	// Bullish: RSI neutral (1), close above SMA50 (2)
	// Bearish: close below SMA20 (1), MACD histogram negative (2)
	// Net score = 0 → HOLD
	r := analysis.Result{
		SMA20:         105,
		SMA50:         95,   // close is above SMA50 → bullish
		RSI14:         50,   // neutral → bullish
		MACD:          0.1,
		MACDSignal:    0.3,
		MACDHistogram: -0.2, // negative → bearish
		LatestClose:   100,  // below SMA20 → bearish
	}
	rec := recommendation.Generate("TSLA", r)
	if rec.Action != recommendation.ActionHold {
		t.Errorf("expected HOLD, got %s (score %d)", rec.Action, rec.Score)
	}
}

func TestGenerate_TickerUppercased(t *testing.T) {
	r := analysis.Result{RSI14: 50, SMA20: 90, LatestClose: 100, MACDHistogram: 0.1}
	rec := recommendation.Generate("aapl", r)
	if rec.Ticker != "AAPL" {
		t.Errorf("ticker should be uppercased, got %q", rec.Ticker)
	}
}

func TestGenerate_ReasonsNotEmpty(t *testing.T) {
	r := analysis.Result{RSI14: 50, SMA20: 90, LatestClose: 100, MACDHistogram: 0.1}
	rec := recommendation.Generate("AAPL", r)
	if len(rec.Reasons) == 0 {
		t.Error("expected at least one reason, got none")
	}
}
