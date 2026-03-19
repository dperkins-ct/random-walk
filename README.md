# random-walk

A Go CLI tool that fetches stock price data, runs technical analysis, and outputs trading recommendations.

## Features

- Fetches daily OHLCV data from the [Alpha Vantage](https://www.alphavantage.co/) API
- Computes technical indicators:
  - **SMA20** – 20-period Simple Moving Average
  - **SMA50** – 50-period Simple Moving Average
  - **RSI14** – 14-period Relative Strength Index (Wilder's smoothing)
  - **MACD** – 12/26-period EMA difference with 9-period signal line
- Outputs a **BUY / SELL / HOLD** recommendation with supporting reasons

## Prerequisites

- Go 1.24+
- A free [Alpha Vantage API key](https://www.alphavantage.co/support/#api-key)

## Installation

```bash
go install github.com/dperkins-ct/random-walk@latest
```

Or build from source:

```bash
git clone https://github.com/dperkins-ct/random-walk.git
cd random-walk
go build -o random-walk .
```

## Usage

```
random-walk analyze <TICKER> [TICKER...] [flags]

Flags:
  --api-key string   Alpha Vantage API key (or set ALPHAVANTAGE_API_KEY env var)
  -v, --verbose      Print detailed indicator values alongside the recommendation
```

### Examples

```bash
# Analyze a single ticker
random-walk analyze AAPL --api-key YOUR_KEY

# Analyze multiple tickers with verbose output
random-walk analyze AAPL MSFT GOOG --api-key YOUR_KEY --verbose

# Use an environment variable for the key
export ALPHAVANTAGE_API_KEY=YOUR_KEY
random-walk analyze TSLA -v
```

### Sample output

```
=== AAPL ===
Recommendation : BUY  (score: +2)
Latest Close   : $189.30
Signals:
  • RSI 48.3 is neutral (30–70)
  • Close $189.30 is above SMA20 $183.75 → short-term uptrend
  • Close $189.30 is above SMA50 $178.50 → medium-term uptrend
  • MACD histogram $0.1234 > 0 → bullish momentum
```

## Architecture

```
random-walk/
├── main.go                            # Entry point
├── cmd/
│   └── root.go                        # CLI commands (cobra)
└── internal/
    ├── api/
    │   └── client.go                  # Alpha Vantage HTTP client
    ├── analysis/
    │   └── analyzer.go                # SMA, RSI, MACD calculations
    └── recommendation/
        └── recommender.go             # BUY/SELL/HOLD logic
```

## Running tests

```bash
go test ./...
```

## License

See [LICENSE](LICENSE).
