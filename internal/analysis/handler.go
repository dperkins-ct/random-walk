package analysis

import (
	"fmt"

	"github.com/dperkins-ct/random-walk/internal/indicators"
)

// Handler orchestrates all analysis indicators.
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
func filterByPeriod(prices []indicators.DailyPrice, period string) ([]indicators.DailyPrice, error) {
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

// Analyze runs all indicators and returns a populated AnalysisResult.
func (h *Handler) Analyze(
	ticker string,
	stockPrices []indicators.DailyPrice,
	marketPrices []indicators.DailyPrice,
	sectorPrices []indicators.DailyPrice, // nil or empty → sector comparison skipped
	overview indicators.Overview,
	period string,
) (indicators.AnalysisResult, error) {
	stock, err := filterByPeriod(stockPrices, period)
	if err != nil {
		return indicators.AnalysisResult{}, err
	}
	market, err := filterByPeriod(marketPrices, period)
	if err != nil {
		return indicators.AnalysisResult{}, err
	}
	var sector []indicators.DailyPrice
	if len(sectorPrices) > 0 {
		sector, _ = filterByPeriod(sectorPrices, period)
	}
	if len(stock) < 50 {
		return indicators.AnalysisResult{}, fmt.Errorf(
			"insufficient price data for %s (%d days available; need at least 50)", ticker, len(stock))
	}

	// Original 6 indicators.
	sharpe := indicators.SharpeRatio(stock, h.annualRiskFreeRate)
	sortino := indicators.SortinoRatio(stock, h.annualRiskFreeRate)
	capmResult := indicators.CAPM(stock, market, h.annualRiskFreeRate)
	maResult := indicators.MovingAverages(stock)
	rsi := indicators.RSI(stock)
	peSignal := indicators.EvaluatePE(overview.PERatio)

	// New 6 indicators.
	bollingerResult, bollingerSig := indicators.BollingerBands(stock)
	obvResult, obvSig := indicators.OnBalanceVolume(stock)
	rsResult, rsSig := indicators.RelativeStrength(stock, market, sector)
	drawdownResult, drawdownSig := indicators.MaxDrawdown(stock)
	varResult, varSig := indicators.ValueAtRisk(stock)
	fundResult := indicators.EvaluateFundamentals(overview)

	sharpeSignal := scoreFloat(sharpe, 1.0, 0.5)
	sortinoSignal := scoreFloat(sortino, 1.5, 0.75)
	capmSignal := scoreCAPM(capmResult)
	maSignalVal := indicators.ModelSignal(maResult.Trend)
	rsiSignalVal := indicators.RSISignal(rsi)
	peSignalVal := indicators.ModelSignal(peSignal)

	const maxScore = 12
	composite := int(sharpeSignal) + int(sortinoSignal) + int(capmSignal) +
		int(maSignalVal) + int(rsiSignalVal) + int(peSignalVal) +
		int(bollingerSig) + int(obvSig) + int(rsSig) +
		int(drawdownSig) + int(varSig) + int(fundResult.Combined)

	recommendation, reasons := recommend(
		composite, sharpe, sortino, capmResult, maResult, rsi, peSignal, overview.PERatio,
		bollingerResult, bollingerSig,
		obvResult, obvSig,
		rsResult, rsSig,
		drawdownResult, drawdownSig,
		varResult, varSig,
		fundResult,
	)

	return indicators.AnalysisResult{
		Ticker:       ticker,
		Name:         overview.Name,
		Sector:       overview.Sector,
		SharpeRatio:  sharpe,
		SortinoRatio: sortino,
		CAP:          capmResult,
		MA:           maResult,
		RSI:          rsi,
		PERatio:      overview.PERatio,
		PESig:        peSignal,

		Bollinger:    bollingerResult,
		OBV:          obvResult,
		RS:           rsResult,
		Drawdown:     drawdownResult,
		VaR:          varResult,
		Fundamentals: fundResult,

		SharpeSignal:       sharpeSignal,
		SortinoSignal:      sortinoSignal,
		CAPMSignal:         capmSignal,
		MASignalVal:        maSignalVal,
		RSISignal:          rsiSignalVal,
		PESignalVal:        peSignalVal,
		BollingerSignal:    bollingerSig,
		OBVSignal:          obvSig,
		RSSignal:           rsSig,
		DrawdownSignal:     drawdownSig,
		VaRSignal:          varSig,
		FundamentalsSignal: fundResult.Combined,

		CompositeScore: composite,
		MaxScore:       maxScore,
		Recommendation: recommendation,
		Reasons:        reasons,
	}, nil
}

func scoreFloat(val, buyThreshold, sellThreshold float64) indicators.ModelSignal {
	if val >= buyThreshold {
		return indicators.SignalBuy
	}
	if val <= sellThreshold {
		return indicators.SignalSell
	}
	return indicators.SignalHold
}

func scoreCAPM(r indicators.CAPMResult) indicators.ModelSignal {
	if r.Alpha > 0.02 {
		return indicators.SignalBuy
	}
	if r.Alpha < -0.02 {
		return indicators.SignalSell
	}
	return indicators.SignalHold
}

func recommend(
	score int,
	sharpe, sortino float64,
	capm indicators.CAPMResult,
	ma indicators.MAResult,
	rsi float64,
	pe indicators.PESignal,
	peRatio float64,
	bollinger indicators.BollingerResult,
	bollingerSig indicators.ModelSignal,
	obv indicators.OBVResult,
	obvSig indicators.ModelSignal,
	rs indicators.RSResult,
	rsSig indicators.ModelSignal,
	drawdown indicators.DrawdownResult,
	drawdownSig indicators.ModelSignal,
	varResult indicators.VaRResult,
	varSig indicators.ModelSignal,
	fund indicators.FundamentalsResult,
) (indicators.Recommendation, []string) {
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
	case indicators.Bullish:
		reasons = append(reasons, fmt.Sprintf(
			"\u25b2 Moving Averages: BULLISH \u2014 momentum indicators broadly agree\n"+
				"  SMA20 (%.2f) > SMA50 (%.2f): price is above its medium-term trend.\n"+
				"  EMA12 (%.2f) > EMA26 (%.2f): short-term momentum is accelerating.\n"+
				"  MACD (%.4f) > Signal (%.4f): buy pressure building.",
			ma.SMA20, ma.SMA50, ma.EMA12, ma.EMA26, ma.MACD, ma.Signal))
	case indicators.Bearish:
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
	case indicators.Cheap:
		reasons = append(reasons, fmt.Sprintf(
			"\u25b2 P/E Ratio: %.1fx \u2014 attractively valued (< 18 is considered cheap)\n"+
				"  Low P/E can signal genuine undervaluation or a stock that has fallen out of favor.\n"+
				"  Worth investigating whether the discount reflects a temporary headwind or a\n"+
				"  structural issue before treating it as a margin-of-safety opportunity.",
			peRatio))
	case indicators.Expensive:
		reasons = append(reasons, fmt.Sprintf(
			"\u25bc P/E Ratio: %.1fx \u2014 growth-priced (> 30 is considered expensive)\n"+
				"  Investors are paying ~%.0fx trailing earnings, pricing in strong future growth.\n"+
				"  At this multiple, any earnings deceleration or guidance cut can trigger sharp\n"+
				"  multiple compression. Sustainable only if growth execution remains consistent.",
			peRatio, peRatio))
	}

	// Bollinger Bands reason.
	switch bollingerSig {
	case indicators.SignalBuy:
		reasons = append(reasons, fmt.Sprintf(
			"\u25b2 Bollinger Bands: close (%.2f) is BELOW the lower band (%.2f)\n"+
				"  Price has moved more than 2 standard deviations below its 20-day mean (%.2f).\n"+
				"  Historically this signals a mean-reversion opportunity. %%B = %.2f (< 0 = oversold).\n"+
				"  Bandwidth %.4f — %s",
			bollinger.Lower+bollinger.PctB*(bollinger.Upper-bollinger.Lower),
			bollinger.Lower, bollinger.Middle, bollinger.PctB,
			bollinger.Bandwidth,
			sqeezeHint(bollinger.Bandwidth)))
	case indicators.SignalSell:
		reasons = append(reasons, fmt.Sprintf(
			"\u25bc Bollinger Bands: close is ABOVE the upper band (%.2f)\n"+
				"  Price has extended more than 2σ above its 20-day mean (%.2f).\n"+
				"  This overbought condition often precedes consolidation or pullback. %%B = %.2f.",
			bollinger.Upper, bollinger.Middle, bollinger.PctB))
	}

	// OBV reason.
	switch obvSig {
	case indicators.SignalBuy:
		reasons = append(reasons, fmt.Sprintf(
			"\u25b2 On-Balance Volume: positive 20-day slope (+%.0f/day)\n"+
				"  Volume is flowing into the stock on up-days faster than it leaves on down-days.\n"+
				"  Rising OBV while price holds or rises confirms institutional accumulation.",
			obv.Slope))
	case indicators.SignalSell:
		reasons = append(reasons, fmt.Sprintf(
			"\u25bc On-Balance Volume: negative 20-day slope (%.0f/day)\n"+
				"  More volume is leaving on down-days than entering on up-days — a distribution signal.\n"+
				"  Divergence between falling OBV and stable price often precedes price decline.",
			obv.Slope))
	}

	// Relative Strength reason.
	switch rsSig {
	case indicators.SignalBuy:
		reasons = append(reasons, fmt.Sprintf(
			"\u25b2 Relative Strength vs SPY: %.2fx \u2014 outperforming the market\n"+
				"  The stock delivered %.1f%% more total return than SPY over the analysis period.\n"+
				"  Stocks that persistently outperform their benchmark tend to continue doing so.",
			rs.VsSPY, (rs.VsSPY-1)*100))
	case indicators.SignalSell:
		reasons = append(reasons, fmt.Sprintf(
			"\u25bc Relative Strength vs SPY: %.2fx \u2014 underperforming the market\n"+
				"  The stock delivered %.1f%% less total return than SPY over the period.\n"+
				"  Persistent underperformance relative to benchmark is a momentum headwind.",
			rs.VsSPY, (1-rs.VsSPY)*100))
	}

	// Maximum Drawdown reason.
	switch drawdownSig {
	case indicators.SignalBuy:
		reasons = append(reasons, fmt.Sprintf(
			"\u25b2 Max Drawdown: %.1f%% \u2014 contained downside risk\n"+
				"  The worst peak-to-trough decline over the period was modest. Calmar Ratio: %.2f\n"+
				"  (annualized return / max drawdown \u2014 higher is better; \u2265 1 is strong).",
			drawdown.MaxDrawdown*100, drawdown.Calmar))
	case indicators.SignalSell:
		reasons = append(reasons, fmt.Sprintf(
			"\u25bc Max Drawdown: %.1f%% \u2014 severe historical loss\n"+
				"  The stock shed more than 40%% of its value from peak to trough in this period.\n"+
				"  Calmar Ratio: %.2f. Deep drawdowns amplify recovery time and test conviction.",
			drawdown.MaxDrawdown*100, drawdown.Calmar))
	}

	// VaR reason.
	switch varSig {
	case indicators.SignalBuy:
		reasons = append(reasons, fmt.Sprintf(
			"\u25b2 Value at Risk (95%%): %.2f%% daily \u2014 low tail risk\n"+
				"  On the worst 5%% of trading days, losses are expected to be at or below this level.\n"+
				"  Expected Shortfall (CVaR): %.2f%% \u2014 average loss in the worst-case tail.",
			varResult.VaR95*100, varResult.CVaR*100))
	case indicators.SignalSell:
		reasons = append(reasons, fmt.Sprintf(
			"\u25bc Value at Risk (95%%): %.2f%% daily \u2014 high tail risk\n"+
				"  On the worst 5%% of trading days, losses routinely exceed this threshold.\n"+
				"  Expected Shortfall (CVaR): %.2f%% \u2014 significant tail exposure warrants caution.",
			varResult.VaR95*100, varResult.CVaR*100))
	}

	// Fundamentals reason (only when data is available).
	if fund.PEGRatio > 0 || fund.DebtToEquity > 0 {
		switch fund.Combined {
		case indicators.SignalBuy:
			reasons = append(reasons, fmt.Sprintf(
				"\u25b2 Fundamentals: %s\n"+
					"  PEG %.2f (< 1 = growth underpriced) | D/E %.2f (< 0.5 = conservative leverage)\n"+
					"  ROE: %.1f%% \u2014 return on shareholders' equity.",
				fundSummary(fund), fund.PEGRatio, fund.DebtToEquity, fund.ROE*100))
		case indicators.SignalSell:
			reasons = append(reasons, fmt.Sprintf(
				"\u25bc Fundamentals: %s\n"+
					"  PEG %.2f (> 2 = growth premium stretched) | D/E %.2f (> 1.5 = high leverage)\n"+
					"  ROE: %.1f%% \u2014 return on shareholders' equity.",
				fundSummary(fund), fund.PEGRatio, fund.DebtToEquity, fund.ROE*100))
		}
	}

	// Recalibrated thresholds for 12 total signals.
	if score >= 5 {
		return indicators.Buy, reasons
	}
	if score <= -5 {
		return indicators.Sell, reasons
	}
	return indicators.Hold, reasons
}

func sqeezeHint(bw float64) string {
	if bw < 0.05 {
		return "narrow bandwidth \u2014 potential volatility squeeze building"
	}
	return "normal bandwidth"
}

func fundSummary(f indicators.FundamentalsResult) string {
	buys, sells := 0, 0
	if f.PEGSignal == indicators.SignalBuy {
		buys++
	} else if f.PEGSignal == indicators.SignalSell {
		sells++
	}
	if f.DERatioSignal == indicators.SignalBuy {
		buys++
	} else if f.DERatioSignal == indicators.SignalSell {
		sells++
	}
	switch {
	case buys > sells:
		return "broadly healthy balance sheet and valuation"
	case sells > buys:
		return "elevated leverage or stretched valuation"
	default:
		return "mixed fundamental signals"
	}
}
