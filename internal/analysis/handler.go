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

	if sharpe >= 1.0 {
		reasons = append(reasons, fmt.Sprintf("Strong Sharpe Ratio (%.2f >= 1.0)", sharpe))
	} else if sharpe <= 0.5 {
		reasons = append(reasons, fmt.Sprintf("Weak Sharpe Ratio (%.2f <= 0.5)", sharpe))
	}
	if sortino >= 1.5 {
		reasons = append(reasons, fmt.Sprintf("Strong Sortino Ratio (%.2f >= 1.5)", sortino))
	} else if sortino <= 0.75 {
		reasons = append(reasons, fmt.Sprintf("Weak Sortino Ratio (%.2f <= 0.75)", sortino))
	}
	if capm.Alpha > 0.02 {
		reasons = append(reasons, fmt.Sprintf("Jensen's Alpha +%.2f%% (outperforming market this period)", capm.Alpha*100))
	} else if capm.Alpha < -0.02 {
		reasons = append(reasons, fmt.Sprintf("Jensen's Alpha %.2f%% (underperforming market this period)", capm.Alpha*100))
	}
	switch ma.Trend {
	case models.Bullish:
		reasons = append(reasons, "Bullish MA crossover (EMA12 > EMA26, SMA20 > SMA50, MACD > Signal)")
	case models.Bearish:
		reasons = append(reasons, "Bearish MA crossover (EMA12 < EMA26, SMA20 < SMA50, MACD < Signal)")
	}
	switch {
	case rsi < 30:
		reasons = append(reasons, fmt.Sprintf("RSI %.1f -- oversold territory (contrarian buy signal)", rsi))
	case rsi > 70:
		reasons = append(reasons, fmt.Sprintf("RSI %.1f -- overbought territory (contrarian sell signal)", rsi))
	}
	switch pe {
	case models.Cheap:
		reasons = append(reasons, fmt.Sprintf("Attractive valuation (P/E %.1f < 18)", peRatio))
	case models.Expensive:
		reasons = append(reasons, fmt.Sprintf("Stretched valuation (P/E %.1f > 30)", peRatio))
	}

	if score >= 3 {
		return models.Buy, reasons
	}
	if score <= -3 {
		return models.Sell, reasons
	}
	return models.Hold, reasons
}
