package analysis

import (
	"fmt"

	"github.com/dperkins-ct/random-walk/internal/models"
)

// Handler orchestrates all analysis models.
type Handler struct {
	annualRiskFreeRate float64
}

// NewHandler constructs an analysis Handler.
// annualRiskFreeRate: e.g. 0.043 (4.3%)
// The annualMarketReturn parameter was removed; CAPM now derives it from the
// actual SPY return over the analysis period.
func NewHandler(annualRiskFreeRate float64) *Handler {
	return &Handler{annualRiskFreeRate: annualRiskFreeRate}
}

// filterByPeriod returns the trailing N trading days of prices.
// period: "1y" (~252), "2y" (~504), or "5y" (~1260). Prices must be chronological.
func filterByPeriod(prices []models.DailyPrice, period string) ([]models.DailyPrice, error) {
	tradingDays := map[string]int{
		"1y": 252,
		"2y": 504,
		"5y": 1260,
	}
	days, ok := tradingDays[period]
	if !ok {
		return nil, fmt.Errorf("unsupported period %q; use 1y, 2y, or 5y", period)
	}
	if len(prices) <= days {
		return prices, nil
	}
	return prices[len(prices)-days:], nil
}

// Analyze runs all models and returns a populated AnalysisResult.
func (h *Handler) Analyze(
	ticker string,
	stockPrices []models.DailyPrice,
	marketPrices []models.DailyPrice,
	overview models.Overview,
	period string,
) (models.AnalysisResult, error) {
	stock, err := filterByPeriod(stockPrices, period)
	if err != nil {
		return models.AnalysisResult{}, err
	}
	market, err := filterByPeriod(marketPrices, period)
	if err != nil {
		return models.AnalysisResult{}, err
	}
	if len(stock) < 50 {
		return models.AnalysisResult{}, fmt.Errorf(
			"insufficient price data for %s (%d days available; need at least 50)", ticker, len(stock))
	}

	sharpe := models.SharpeRatio(stock, h.annualRiskFreeRate)
	sortino := models.SortinoRatio(stock, h.annualRiskFreeRate)
	capmResult := models.CAPM(stock, market, h.annualRiskFreeRate)
	maResult := models.MovingAverages(stock)
	rsi := models.RSI(stock)
	peSignal := models.EvaluatePE(overview.PERatio)

	sharpeSignal := scoreFloat(sharpe, 1.0, 0.5)
	sortinoSignal := scoreFloat(sortino, 1.5, 0.75)
	capmSignal := scoreCAPM(capmResult)
	maSignalVal := models.ModelSignal(maResult.Trend)
	rsiSignalVal := models.RSISignal(rsi)
	peSignalVal := models.ModelSignal(peSignal)

	const maxScore = 6
	composite := int(sharpeSignal) + int(sortinoSignal) + int(capmSignal) +
		int(maSignalVal) + int(rsiSignalVal) + int(peSignalVal)

	recommendation, reasons := recommend(
		composite, sharpe, sortino, capmResult, maResult, rsi, peSignal, overview.PERatio)

	return models.AnalysisResult{
		Ticker:         ticker,
		Name:           overview.Name,
		Sector:         overview.Sector,
		SharpeRatio:    sharpe,
		SortinoRatio:   sortino,
		CAP:            capmResult,
		MA:             maResult,
		RSI:            rsi,
		PERatio:        overview.PERatio,
		PESig:          peSignal,
		SharpeSignal:   sharpeSignal,
		SortinoSignal:  sortinoSignal,
		CAPMSignal:     capmSignal,
		MASignalVal:    maSignalVal,
		RSISignal:      rsiSignalVal,
		PESignalVal:    peSignalVal,
		CompositeScore: composite,
		MaxScore:       maxScore,
		Recommendation: recommendation,
		Reasons:        reasons,
	}, nil
}

func scoreFloat(val, buyThreshold, sellThreshold float64) models.ModelSignal {
	if val >= buyThreshold {
		return models.SignalBuy
	}
	if val <= sellThreshold {
		return models.SignalSell
	}
	return models.SignalHold
}

func scoreCAPM(r models.CAPMResult) models.ModelSignal {
	if r.Alpha > 0.02 {
		return models.SignalBuy
	}
	if r.Alpha < -0.02 {
		return models.SignalSell
	}
	return models.SignalHold
}

func recommend(
	score int,
	sharpe, sortino float64,
	capm models.CAPMResult,
	ma models.MAResult,
	rsi float64,
	pe models.PESignal,
	peRatio float64,
) (models.Recommendation, []string) {
	var reasons []string

	switch {
	case sharpe >= 1.0:
		reasons = append(reasons, fmt.Sprintf(
			"\u25b2 Sharpe Ratio: %.4f \u2014 strong risk-adjusted returns (threshold \u2265 1.0)\n"+
				"  For every unit of total risk, this stock delivers solid excess returns above the\n"+
				"  risk-free rate. A Sharpe above 1.0 signals that volatility is being well rewarded.",
			sharpe))
	case sharpe <= 0.5:
		reasons = append(reasons, fmt.Sprintf(
			"\u25bc Sharpe Ratio: %.4f \u2014 weak risk-adjusted returns (0.5\u20131.0 acceptable, \u2265 1.0 strong)\n"+
				"  Not enough excess return is being generated relative to the total volatility carried.\n"+
				"  Most quality large-cap equities sustain 0.5\u20131.0; above 1.0 is considered strong.",
			sharpe))
	}

	switch {
	case sortino >= 1.5:
		reasons = append(reasons, fmt.Sprintf(
			"\u25b2 Sortino Ratio: %.4f \u2014 strong downside-adjusted returns (threshold \u2265 1.5)\n"+
				"  Unlike Sharpe, Sortino only penalizes harmful (downward) volatility. This high value\n"+
				"  means gains are materially outpacing drawdowns \u2014 the volatility is mostly to the upside.",
			sortino))
	case sortino <= 0.75:
		reasons = append(reasons, fmt.Sprintf(
			"\u25bc Sortino Ratio: %.4f \u2014 weak downside-adjusted returns (0.75\u20131.5 acceptable, \u2265 1.5 strong)\n"+
				"  Sortino only counts negative swings as risk. A value this low signals that the stock's\n"+
				"  down-moves are disproportionately large relative to its gains \u2014 a poor risk profile.",
			sortino))
	}

	switch {
	case capm.Alpha > 0.02:
		reasons = append(reasons, fmt.Sprintf(
			"\u25b2 Jensen's Alpha: +%.2f%% \u2014 beating the risk-adjusted market expectation\n"+
				"  CAPM predicted +%.2f%% (beta %.2f against SPY's actual +%.2f%%). The stock\n"+
				"  beat that by %.2f%%, suggesting idiosyncratic strengths beyond market exposure.",
			capm.Alpha*100, capm.ExpectedReturn*100, capm.Beta, capm.ActualMarketReturn*100, capm.Alpha*100))
	case capm.Alpha < -0.02:
		reasons = append(reasons, fmt.Sprintf(
			"\u25bc Jensen's Alpha: %.2f%% \u2014 below the risk-adjusted market expectation\n"+
				"  CAPM predicted +%.2f%% (beta %.2f against SPY's actual +%.2f%%). The stock\n"+
				"  fell short by %.2f%%, absorbing excess market sensitivity without commensurate reward.",
			capm.Alpha*100, capm.ExpectedReturn*100, capm.Beta, capm.ActualMarketReturn*100, -capm.Alpha*100))
	}

	switch ma.Trend {
	case models.Bullish:
		reasons = append(reasons, fmt.Sprintf(
			"\u25b2 Moving Averages: BULLISH \u2014 momentum indicators broadly agree\n"+
				"  SMA20 (%.2f) > SMA50 (%.2f): price is above its medium-term trend.\n"+
				"  EMA12 (%.2f) > EMA26 (%.2f): short-term momentum is accelerating.\n"+
				"  MACD (%.4f) > Signal (%.4f): buy pressure building.",
			ma.SMA20, ma.SMA50, ma.EMA12, ma.EMA26, ma.MACD, ma.Signal))
	case models.Bearish:
		reasons = append(reasons, fmt.Sprintf(
			"\u25bc Moving Averages: BEARISH \u2014 momentum indicators broadly agree\n"+
				"  SMA20 (%.2f) < SMA50 (%.2f): price is below its medium-term trend.\n"+
				"  EMA12 (%.2f) < EMA26 (%.2f): short-term momentum is declining.\n"+
				"  MACD (%.4f) < Signal (%.4f): sell pressure building.",
			ma.SMA20, ma.SMA50, ma.EMA12, ma.EMA26, ma.MACD, ma.Signal))
	}

	switch {
	case rsi < 30:
		reasons = append(reasons, fmt.Sprintf(
			"\u25b2 RSI (14): %.1f \u2014 oversold territory (< 30 is a contrarian buy signal)\n"+
				"  The stock has been sold aggressively enough that a mean-reversion bounce is\n"+
				"  historically more likely from here. Oversold can persist in sustained downtrends;\n"+
				"  treat as a supporting signal alongside trend and fundamental analysis.",
			rsi))
	case rsi > 70:
		reasons = append(reasons, fmt.Sprintf(
			"\u25bc RSI (14): %.1f \u2014 overbought territory (> 70 is a contrarian sell signal)\n"+
				"  The stock has rallied sharply and near-term pullback risk is elevated. Overbought\n"+
				"  readings often precede consolidation or correction, particularly when paired\n"+
				"  with stretched valuations or deteriorating MACD momentum.",
			rsi))
	}

	switch pe {
	case models.Cheap:
		reasons = append(reasons, fmt.Sprintf(
			"\u25b2 P/E Ratio: %.1fx \u2014 attractively valued (< 18 is considered cheap)\n"+
				"  Low P/E can signal genuine undervaluation or a stock that has fallen out of favor.\n"+
				"  Worth investigating whether the discount reflects a temporary headwind or a\n"+
				"  structural issue before treating it as a margin-of-safety opportunity.",
			peRatio))
	case models.Expensive:
		reasons = append(reasons, fmt.Sprintf(
			"\u25bc P/E Ratio: %.1fx \u2014 growth-priced (> 30 is considered expensive)\n"+
				"  Investors are paying ~%.0fx trailing earnings, pricing in strong future growth.\n"+
				"  At this multiple, any earnings deceleration or guidance cut can trigger sharp\n"+
				"  multiple compression. Sustainable only if growth execution remains consistent.",
			peRatio, peRatio))
	}

	if score >= 3 {
		return models.Buy, reasons
	}
	if score <= -3 {
		return models.Sell, reasons
	}
	return models.Hold, reasons
}
