# random-walk

A Go CLI tool that fetches historical stock data from Alpha Vantage, runs multiple quantitative finance models, and outputs a color-coded terminal report with a **BUY / HOLD / SELL** recommendation.

## Features

### Data
- Fetches **daily adjusted prices** via the Alpha Vantage `TIME_SERIES_DAILY_ADJUSTED` endpoint
- Fetches **fundamental data** (P/E ratio, sector) via the Alpha Vantage `OVERVIEW` endpoint
- **CSV caching** at `~/.random-walk/cache/` — re-uses today's data to avoid burning API quota
- Uses **SPY** as the market benchmark for CAPM (also cached)

### Models
| Model | What it measures |
|---|---|
| **Sharpe Ratio** | Annualized risk-adjusted return (excess return / total volatility × √252) |
| **Sortino Ratio** | Like Sharpe, but penalizes only downside volatility |
| **CAPM / Beta** | Market sensitivity (Beta), expected return, and Jensen's Alpha |
| **SMA / EMA Crossovers** | SMA20/50 and EMA12/26 trend direction; MACD vs signal line |
| **P/E Valuation** | Classifies P/E as Cheap (< 18) / Fair / Expensive (> 30) vs S&P 500 median |

### Composite Scoring
Each model votes **+1 (bullish)**, **0 (neutral)**, or **−1 (bearish)**.  
Score ≥ 3 → **BUY** · Score ≤ −3 → **SELL** · Otherwise → **HOLD**

## Prerequisites

- Go 1.24+
- A free [Alpha Vantage API key](https://www.alphavantage.co/support/#api-key) (25 requests/day on the free tier)

## Installation

```bash
git clone https://github.com/dperkins-ct/random-walk.git
cd random-walk
go build -o random-walk ./cmd
```

## Usage

```
random-walk <TICKER> [flags]

Flags:
  --api-key string        Alpha Vantage API key (or set ALPHAVANTAGE_API_KEY env var)
  --period string         Historical lookback: 1y | 2y | 5y  (default "1y")
  --risk-free-rate float  Annual risk-free rate as decimal, e.g. 0.043  (default 0.043)
  --market-return float   Expected annual market return as decimal, e.g. 0.10  (default 0.10)
```

### Examples

```bash
# Analyze AAPL over the past year
random-walk AAPL --api-key YOUR_KEY

# Use an environment variable for the key
export ALPHAVANTAGE_API_KEY=YOUR_KEY
random-walk MSFT --period 2y

# Override risk-free rate (e.g. current 3-month T-bill)
random-walk NVDA --risk-free-rate 0.052
```

### Sample output

```
────────────────────────────────────────────────────────────
  AAPL  Apple Inc.
  Sector: TECHNOLOGY
────────────────────────────────────────────────────────────
  METRICS
  Sharpe Ratio (annualized):     1.4231
  Sortino Ratio (annualized):    2.1089
  Beta:                          1.2341
  CAPM Expected Return:          14.84%
  Jensen's Alpha:                +3.21%
  SMA20:                         183.4200
  SMA50:                         179.8800
  EMA12:                         184.1100
  EMA26:                         181.3300
  MACD:                          2.7800
  MACD Signal Line:              2.1200
  P/E Ratio:                     28.50

  SIGNALS
  Sharpe:                        BUY  (+1)
  Sortino:                       BUY  (+1)
  CAPM (Jensen's Alpha):         BUY  (+1)
  Moving Averages:               BUY  (+1)
  P/E Valuation:                 HOLD ( 0)
  Composite Score:               4 / 5

────────────────────────────────────────────────────────────
  RECOMMENDATION:  BUY

  Supporting Reasons:
  • Strong Sharpe Ratio (1.42 > 1.0)
  • Strong Sortino Ratio (2.11 > 1.5)
  • Positive Jensen's Alpha (+3.21%)
  • Bullish MA crossover (EMA12 > EMA26, SMA20 > SMA50, MACD > Signal)
────────────────────────────────────────────────────────────
```

## Project structure

```
random-walk/
├── cmd/
│   └── main.go                  # Entry point: main() → run() → os.Exit only here
├── internal/
│   ├── api/
│   │   └── handler.go           # Alpha Vantage HTTP calls (FetchPrices, FetchOverview)
│   ├── analysis/
│   │   └── handler.go           # Orchestrates models, composite scoring
│   ├── cache/
│   │   └── csv.go               # CSV read/write + same-day freshness check
│   ├── models/
│   │   ├── types.go             # Shared data types (DailyPrice, Overview, AnalysisResult…)
│   │   ├── sharpe.go            # Sharpe Ratio
│   │   ├── sortino.go           # Sortino Ratio
│   │   ├── capm.go              # Beta, Expected Return, Jensen's Alpha
│   │   ├── moving_avg.go        # SMA/EMA crossovers, MACD
│   │   └── pe.go                # P/E valuation signal
│   └── output/
│       └── terminal.go          # ANSI-colored terminal report
└── go.mod
```

## API quota notes

The free Alpha Vantage tier allows **25 requests per day**.  
A fresh run of `random-walk` uses **3 requests** (stock prices, SPY prices, overview).  
Subsequent runs on the same calendar day use **0 requests** (served from CSV cache).

## Running tests

```bash
go test ./...
```

## License

See [LICENSE](LICENSE).
