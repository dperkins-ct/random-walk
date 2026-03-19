package output

import (
	"fmt"
	"strings"

	"github.com/dperkins-ct/random-walk/internal/indicators"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

func green(s string) string  { return colorGreen + s + colorReset }
func yellow(s string) string { return colorYellow + s + colorReset }
func red(s string) string    { return colorRed + s + colorReset }
func cyan(s string) string   { return colorCyan + s + colorReset }
func bold(s string) string   { return colorBold + s + colorReset }
func dim(s string) string    { return colorDim + s + colorReset }

const lineWidth = 72

func divider() string { return strings.Repeat("\u2500", lineWidth) }

// Print renders a full ANSI-colored analysis report to stdout.
func Print(result indicators.AnalysisResult) {
	fmt.Println()
	fmt.Println(bold(cyan(divider())))

	name := result.Name
	if name == "" {
		name = result.Ticker
	}
	fmt.Printf("  %s  %s\n", bold(cyan(result.Ticker)), name)
	if result.Sector != "" {
		fmt.Printf("  Sector: %s\n", result.Sector)
	}
	fmt.Println(bold(cyan(divider())))
	fmt.Println()

	// --- Metrics table ---------------------------------------------------
	// Three columns: METRIC (28) | VALUE (16) | REFERENCE GUIDE
	// Ref column starts at visible col 49.  All ref strings are <= 31 visible
	// chars so they never wrap on an 80-column terminal.
	const labelW = 28
	const valueW = 16
	fmt.Println("  " + bold("METRIC") + strings.Repeat(" ", labelW-6) +
		bold("VALUE") + strings.Repeat(" ", valueW-5) + dim("REFERENCE GUIDE"))
	fmt.Println("  " + strings.Repeat("\u00b7", lineWidth-2))

	// Sharpe  (25 visible chars)
	printRow("Sharpe Ratio (annualized)",
		colorSharpe(result.SharpeRatio),
		fmt.Sprintf("%.4f", result.SharpeRatio),
		dim("< 0.5 weak | \u2265 1.0 strong"))

	// Sortino  (26 visible chars)
	printRow("Sortino Ratio (annualized)",
		colorSortino(result.SortinoRatio),
		fmt.Sprintf("%.4f", result.SortinoRatio),
		dim("< 0.75 weak | \u2265 1.5 strong"))

	// Beta  (31 visible chars)
	printRow("Beta (market sensitivity)",
		colorBeta(result.CAP.Beta),
		fmt.Sprintf("%.4f", result.CAP.Beta),
		dim("low <0.8 | moderate | high >1.5"))

	// Market Return  (30 visible chars)
	mktRaw := pct(result.CAP.ActualMarketReturn)
	printRow("Market Return (SPY, period)", yellow(mktRaw), mktRaw,
		dim("actual S&P 500 return (period)"))

	// CAPM Expected Return  (29 visible chars)
	capmRaw := pct(result.CAP.ExpectedReturn)
	printRow("CAPM Expected Return", yellow(capmRaw), capmRaw,
		dim("beta-adjusted return forecast"))

	// Jensen's Alpha  (29 visible chars)
	alphaRaw := pct(result.CAP.Alpha)
	printRow("Jensen's Alpha",
		colorAlpha(result.CAP.Alpha), alphaRaw,
		dim("< -2% lagging | > +2% beating"))

	fmt.Println()

	// Moving Averages -- each row shows a dynamic contextual label that
	// states what the current comparison means (not a generic if/then guide).
	maColor := colorMAFn(result.MA.Trend)

	smaRaw := fmt.Sprintf("%.2f / %.2f", result.MA.SMA20, result.MA.SMA50)
	var smaRef string
	if result.MA.SMA20 > result.MA.SMA50 {
		smaRef = dim("20 > 50 \u2192 above trend (bullish)") // 31 chars
	} else {
		smaRef = dim("20 < 50 \u2192 below trend (bearish)") // 31 chars
	}
	printRow("SMA 20 / SMA 50", maColor(smaRaw), smaRaw, smaRef)

	emaRaw := fmt.Sprintf("%.2f / %.2f", result.MA.EMA12, result.MA.EMA26)
	var emaRef string
	if result.MA.EMA12 > result.MA.EMA26 {
		emaRef = dim("12 > 26 \u2192 bullish momentum") // 25 chars
	} else {
		emaRef = dim("12 < 26 \u2192 bearish momentum") // 25 chars
	}
	printRow("EMA 12 / EMA 26", maColor(emaRaw), emaRaw, emaRef)

	// MACD at %.2f precision so the combined value stays <= 15 chars
	macdRaw := fmt.Sprintf("%.2f / %.2f", result.MA.MACD, result.MA.Signal)
	var macdRef string
	if result.MA.MACD > result.MA.Signal {
		macdRef = dim("MACD > Signal \u2192 buy pressure") // 28 chars
	} else {
		macdRef = dim("MACD < Signal \u2192 sell pressure") // 29 chars
	}
	printRow("MACD / Signal Line", maColor(macdRaw), macdRaw, macdRef)

	fmt.Println()

	// RSI  (31 visible chars)
	rsiRaw := fmt.Sprintf("%.2f", result.RSI)
	printRow("RSI (14)", colorRSIVal(result.RSI), rsiRaw,
		dim("< 30 oversold | > 70 overbought"))

	// P/E  (27 visible chars)
	peRaw := "N/A"
	if result.PERatio > 0 {
		peRaw = fmt.Sprintf("%.2f", result.PERatio)
	}
	printRow("P/E Ratio", colorPE(result.PERatio), peRaw,
		dim("< 18 cheap | > 30 expensive"))

	fmt.Println()

	// --- Bollinger Bands --------------------------------------------------
	fmt.Println(bold("  VOLATILITY & VOLUME"))
	fmt.Println("  " + strings.Repeat("\u00b7", lineWidth-2))

	bbRaw := fmt.Sprintf("%.2f / %.2f / %.2f", result.Bollinger.Lower, result.Bollinger.Middle, result.Bollinger.Upper)
	var bbRef string
	switch {
	case result.BollingerSignal == indicators.SignalBuy:
		bbRef = dim("price below lower band \u2192 oversold")
	case result.BollingerSignal == indicators.SignalSell:
		bbRef = dim("price above upper band \u2192 overbought")
	default:
		bbRef = dim(fmt.Sprintf("%%B=%.2f bw=%.4f", result.Bollinger.PctB, result.Bollinger.Bandwidth))
	}
	printRow("Bollinger (L/M/U)", colorSignal(result.BollingerSignal)(bbRaw), bbRaw, bbRef)

	// OBV
	obvRaw := fmt.Sprintf("%d", result.OBV.OBV)
	if len(obvRaw) > 15 {
		obvRaw = fmt.Sprintf("%.3eM", float64(result.OBV.OBV)/1e6)
	}
	var obvRef string
	if result.OBV.Slope > 0 {
		obvRef = dim(fmt.Sprintf("slope +%.0f/day \u2192 accumulation", result.OBV.Slope))
	} else {
		obvRef = dim(fmt.Sprintf("slope %.0f/day \u2192 distribution", result.OBV.Slope))
	}
	printRow("OBV (20d slope)", colorSignal(result.OBVSignal)(obvRaw), obvRaw, obvRef)

	fmt.Println()

	// --- Market Context ---------------------------------------------------
	fmt.Println(bold("  MARKET CONTEXT"))
	fmt.Println("  " + strings.Repeat("\u00b7", lineWidth-2))

	rsRaw := fmt.Sprintf("%.3f", result.RS.VsSPY)
	var rsRef string
	switch result.RSSignal {
	case indicators.SignalBuy:
		rsRef = dim(fmt.Sprintf("> 1.10 \u2192 outperforming SPY by +%.1f%%", (result.RS.VsSPY-1)*100))
	case indicators.SignalSell:
		rsRef = dim(fmt.Sprintf("< 0.90 \u2192 underperforming SPY by %.1f%%", (1-result.RS.VsSPY)*100))
	default:
		rsRef = dim("within \u00b110%% of SPY return")
	}
	printRow("Rel Strength vs SPY", colorSignal(result.RSSignal)(rsRaw), rsRaw, rsRef)

	if result.RS.VsSector != 0 {
		rsSecRaw := fmt.Sprintf("%.3f", result.RS.VsSector)
		var rsSecRef string
		if result.RS.VsSector > 1 {
			rsSecRef = dim(fmt.Sprintf("outperforming sector ETF by +%.1f%%", (result.RS.VsSector-1)*100))
		} else {
			rsSecRef = dim(fmt.Sprintf("underperforming sector ETF by %.1f%%", (1-result.RS.VsSector)*100))
		}
		printRow("Rel Strength vs Sector", yellow(rsSecRaw), rsSecRaw, rsSecRef)
	}

	fmt.Println()

	// --- Risk ---------------------------------------------------------------
	fmt.Println(bold("  RISK"))
	fmt.Println("  " + strings.Repeat("\u00b7", lineWidth-2))

	ddRaw := fmt.Sprintf("%.2f%%", result.Drawdown.MaxDrawdown*100)
	ddRef := dim(fmt.Sprintf("Calmar %.2f | > 40%% severe", result.Drawdown.Calmar))
	printRow("Max Drawdown", colorSignal(result.DrawdownSignal)(ddRaw), ddRaw, ddRef)

	varRaw := fmt.Sprintf("%.2f%%", result.VaR.VaR95*100)
	varRef := dim(fmt.Sprintf("CVaR %.2f%% | < -3%% high tail risk", result.VaR.CVaR*100))
	printRow("VaR 95% (daily)", colorSignal(result.VaRSignal)(varRaw), varRaw, varRef)

	fmt.Println()

	// --- Fundamentals (only when data is available) -----------------------
	if result.Fundamentals.PEGRatio > 0 || result.Fundamentals.PriceToBook > 0 {
		fmt.Println(bold("  FUNDAMENTALS"))
		fmt.Println("  " + strings.Repeat("\u00b7", lineWidth-2))

		pegRaw := "N/A"
		if result.Fundamentals.PEGRatio > 0 {
			pegRaw = fmt.Sprintf("%.2f", result.Fundamentals.PEGRatio)
		}
		pegRef := dim("< 1 undervalued | 1-2 fair | > 2 stretched")
		printRow("PEG Ratio", colorPEG(result.Fundamentals.PEGRatio), pegRaw, pegRef)

		pbRaw := "N/A"
		if result.Fundamentals.PriceToBook > 0 {
			pbRaw = fmt.Sprintf("%.2f", result.Fundamentals.PriceToBook)
		}
		pbRef := dim("< 1 below book | 1-4 normal | > 4 premium")
		printRow("Price-to-Book (P/B)", colorPB(result.Fundamentals.PriceToBook), pbRaw, pbRef)

		if result.Fundamentals.ROE != 0 {
			roeRaw := fmt.Sprintf("%.1f%%", result.Fundamentals.ROE*100)
			printRow("Return on Equity", yellow(roeRaw), roeRaw, dim("profitability of shareholder equity"))
		}

		fmt.Println()
	}

	// --- Signals ----------------------------------------------------------
	fmt.Println(bold("  SIGNALS"))
	printSig("Sharpe:", signalStr(result.SharpeSignal))
	printSig("Sortino:", signalStr(result.SortinoSignal))
	printSig("CAPM (Jensen's Alpha):", signalStr(result.CAPMSignal))
	printSig("Moving Averages:", signalStr(result.MASignalVal))
	printSig("RSI (14):", signalStr(result.RSISignal))
	printSig("P/E Valuation:", signalStr(result.PESignalVal))
	printSig("Bollinger Bands:", signalStr(result.BollingerSignal))
	printSig("On-Balance Volume:", signalStr(result.OBVSignal))
	printSig("Relative Strength:", signalStr(result.RSSignal))
	printSig("Max Drawdown:", signalStr(result.DrawdownSignal))
	printSig("Value at Risk:", signalStr(result.VaRSignal))
	if result.Fundamentals.PEGRatio > 0 || result.Fundamentals.PriceToBook > 0 {
		printSig("Fundamentals (PEG/D/E):", signalStr(result.FundamentalsSignal))
	}

	fmt.Printf("  %-28s %.1f  %s\n",
		"Composite Score:", result.CompositeScore,
		dim(fmt.Sprintf("(range \u2212%.1f to +%.1f | BUY \u2265 6.0 | SELL \u2264 \u22126.0)", result.MaxScore, result.MaxScore)))
	fmt.Println()

	// --- Recommendation ---------------------------------------------------
	fmt.Println(bold(cyan(divider())))
	recColor := recColorFn(result.Recommendation)
	fmt.Printf("  RECOMMENDATION:  %s\n", bold(recColor(string(result.Recommendation))))
	fmt.Println()

	if len(result.Reasons) > 0 {
		fmt.Println(bold("  Analysis:"))
		fmt.Println()
		for i, r := range result.Reasons {
			if i > 0 {
				fmt.Println()
			}
			lines := strings.Split(r, "\n")
			for j, line := range lines {
				if j == 0 {
					// Color summary line based on leading signal indicator.
					switch {
					case strings.HasPrefix(line, "\u25b2"):
						fmt.Printf("  %s\n", green(line))
					case strings.HasPrefix(line, "\u25bc"):
						fmt.Printf("  %s\n", red(line))
					default:
						fmt.Printf("  %s\n", line)
					}
				} else {
					// Detail lines dimmed to visually subordinate them.
					fmt.Printf("  %s\n", dim(line))
				}
			}
		}
	}
	fmt.Println()
	fmt.Println(bold(cyan(divider())))
	fmt.Println()
}

// printRow prints one metrics row with three columns.
// coloredValue contains ANSI escape codes; rawValue is the plain text used for
// padding so ANSI bytes don't misalign the reference column.
func printRow(label, coloredValue, rawValue, ref string) {
	const valueW = 16
	pad := valueW - len(rawValue)
	if pad < 1 {
		pad = 1
	}
	fmt.Printf("  %-28s %s%s %s\n", label, coloredValue, strings.Repeat(" ", pad), ref)
}

func printSig(label, value string) {
	fmt.Printf("  %-28s %s\n", label, value)
}

// pct formats a decimal fraction as a percentage, prefixing + for positive values.
func pct(f float64) string {
	if f >= 0 {
		return fmt.Sprintf("+%.2f%%", f*100)
	}
	return fmt.Sprintf("%.2f%%", f*100)
}

// --- Value coloring helpers -----------------------------------------------

func colorSharpe(v float64) string {
	s := fmt.Sprintf("%.4f", v)
	switch {
	case v >= 1.0:
		return green(s)
	case v >= 0.5:
		return yellow(s)
	default:
		return red(s)
	}
}

func colorSortino(v float64) string {
	s := fmt.Sprintf("%.4f", v)
	switch {
	case v >= 1.5:
		return green(s)
	case v >= 0.75:
		return yellow(s)
	default:
		return red(s)
	}
}

func colorBeta(v float64) string {
	s := fmt.Sprintf("%.4f", v)
	switch {
	case v > 1.5:
		return red(s)
	case v < 0.8:
		return green(s)
	default:
		return yellow(s)
	}
}

func colorAlpha(v float64) string {
	s := pct(v)
	switch {
	case v > 0.02:
		return green(s)
	case v < -0.02:
		return red(s)
	default:
		return yellow(s)
	}
}

func colorMAFn(trend indicators.MASignal) func(string) string {
	switch trend {
	case indicators.Bullish:
		return green
	case indicators.Bearish:
		return red
	default:
		return yellow
	}
}

func colorRSIVal(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	switch {
	case v < 30:
		return green(s) // oversold: contrarian buy signal
	case v > 70:
		return red(s) // overbought: contrarian sell signal
	default:
		return yellow(s)
	}
}

func colorPE(v float64) string {
	if v <= 0 {
		return yellow("N/A")
	}
	s := fmt.Sprintf("%.2f", v)
	switch {
	case v < 18:
		return green(s)
	case v <= 30:
		return yellow(s)
	default:
		return red(s)
	}
}

func signalStr(s indicators.ModelSignal) string {
	switch s {
	case indicators.SignalBuy:
		return green("BUY  (+1)")
	case indicators.SignalSell:
		return red("SELL (-1)")
	default:
		return yellow("HOLD ( 0)")
	}
}

func recColorFn(r indicators.Recommendation) func(string) string {
	switch r {
	case indicators.Buy:
		return green
	case indicators.Sell:
		return red
	default:
		return yellow
	}
}

// colorSignal returns a coloring function based on a ModelSignal.
func colorSignal(s indicators.ModelSignal) func(string) string {
	switch s {
	case indicators.SignalBuy:
		return green
	case indicators.SignalSell:
		return red
	default:
		return yellow
	}
}

func colorPEG(v float64) string {
	if v <= 0 {
		return yellow("N/A")
	}
	s := fmt.Sprintf("%.2f", v)
	switch {
	case v < 1.0:
		return green(s)
	case v <= 2.0:
		return yellow(s)
	default:
		return red(s)
	}
}

func colorPB(v float64) string {
	if v <= 0 {
		return yellow("N/A")
	}
	s := fmt.Sprintf("%.2f", v)
	switch {
	case v < 1.0:
		return green(s)
	case v <= 4.0:
		return yellow(s)
	default:
		return red(s)
	}
}
