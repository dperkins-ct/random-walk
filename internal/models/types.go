package models

// DailyPrice holds OHLCV data for a single trading day.
type DailyPrice struct {
	Date     string
	Open     float64
	High     float64
	Low      float64
	Close    float64
	AdjClose float64
	Volume   int64
}

// Overview holds fundamental data from the Alpha Vantage OVERVIEW endpoint.
type Overview struct {
	Symbol    string
	Name      string
	Sector    string
	PERatio   float64
	ForwardPE float64
}

// CAPMResult holds the output of the CAPM model.
type CAPMResult struct {
	Beta               float64
	ActualMarketReturn float64 // annualised, realised over the period
	ExpectedReturn     float64 // annualised, CAPM-predicted
	Alpha              float64 // Jensen's alpha (actual - expected)
}

// MASignal represents a directional signal from moving average analysis.
type MASignal int

const (
	Bullish MASignal = 1
	Neutral MASignal = 0
	Bearish MASignal = -1
)

// MAResult holds computed moving average values and the derived signal.
type MAResult struct {
	SMA20  float64
	SMA50  float64
	EMA12  float64
	EMA26  float64
	MACD   float64
	Signal float64
	Trend  MASignal
}

// PESignal classifies whether a stock's P/E is cheap, fair, or expensive.
type PESignal int

const (
	Cheap     PESignal = 1
	Fair      PESignal = 0
	Expensive PESignal = -1
)

// ModelSignal is a generic +1/0/-1 vote from any model.
type ModelSignal int

const (
	SignalBuy  ModelSignal = 1
	SignalHold ModelSignal = 0
	SignalSell ModelSignal = -1
)

// Recommendation is the final BUY/HOLD/SELL output.
type Recommendation string

const (
	Buy  Recommendation = "BUY"
	Hold Recommendation = "HOLD"
	Sell Recommendation = "SELL"
)

// AnalysisResult holds the full output of all models plus the final recommendation.
type AnalysisResult struct {
	Ticker string
	Name   string
	Sector string

	SharpeRatio  float64
	SortinoRatio float64
	CAP          CAPMResult
	MA           MAResult
	RSI          float64
	PERatio      float64
	PESig        PESignal

	SharpeSignal  ModelSignal
	SortinoSignal ModelSignal
	CAPMSignal    ModelSignal
	MASignalVal   ModelSignal
	RSISignal     ModelSignal
	PESignalVal   ModelSignal

	CompositeScore int
	MaxScore       int
	Recommendation Recommendation
	Reasons        []string
}
