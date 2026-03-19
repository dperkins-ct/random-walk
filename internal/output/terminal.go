package output

import (
	"fmt"
	"strings"

	"github.com/dperkins-ct/random-walk/internal/models"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

func green(s string) string  { return colorGreen + s + colorReset }
func yellow(s string) string { return colorYellow + s + colorReset }
func red(s string) string    { return colorRed + s + colorReset }
func cyan(s string) string   { return colorCyan + s + colorReset }
func bold(s string) string   { return colorBold + s + colorReset }

func divider() string { return strings.Repeat("─", 60) }

// Print renders a full ANSI-colored analysis report to stdout.
func Print(result models.AnalysisResult) {
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

	// --- Metrics ----------------------------------------------------------
	fmt.Println(bold("  METRICS"))
	printMetric("Sharpe Ratio (annualized):", fmt.Sprintf("%.4f", result.SharpeRatio))
	printMetric("Sortino Ratio (annualized):", fmt.Sprintf("%.4f", result.SortinoRatio))
	printMetric("Beta:", fmt.Sprintf("%.4f", result.CAP.Beta))
	printMetric("Market Return (period):", pct(result.CAP.ActualMarketReturn))
	printMetric("CAPM Expected Return:", pct(result.CAP.ExpectedReturn))
	printMetric("Jensen's Alpha:", pct(result.CAP.Alpha))
	printMetric("SMA20:", fmt.Sprintf("%.4f", result.MA.SMA20))
	printMetric("SMA50:", fmt.Sprintf("%.4f", result.MA.SMA50))
	printMetric("EMA12:", fmt.Sprintf("%.4f", result.MA.EMA12))
	printMetric("EMA26:", fmt.Sprintf("%.4f", result.MA.EMA26))
	printMetric("MACD:", fmt.Sprintf("%.4f", result.MA.MACD))
	printMetric("MACD Signal Line:", fmt.Sprintf("%.4f", result.MA.Signal))
	rsiStr := fmt.Sprintf("%.2f", result.RSI)
	switch {
	case result.RSI < 30:
		rsiStr += "  (oversold)"
	case result.RSI > 70:
		rsiStr += "  (overbought)"
	}
	printMetric("RSI (14):", rsiStr)
	peStr := "N/A"
	if result.PERatio > 0 {
		peStr = fmt.Sprintf("%.2f", result.PERatio)
	}
	printMetric("P/E Ratio:", peStr)
	fmt.Println()

	// --- Signals ----------------------------------------------------------
	fmt.Println(bold("  SIGNALS"))
	printMetric("Sharpe:", signalStr(result.SharpeSignal))
	printMetric("Sortino:", signalStr(result.SortinoSignal))
	printMetric("CAPM (Jensen's Alpha):", signalStr(result.CAPMSignal))
	printMetric("Moving Averages:", signalStr(result.MASignalVal))
	printMetric("RSI (14):", signalStr(result.RSISignal))
	printMetric("P/E Valuation:", signalStr(result.PESignalVal))
	fmt.Printf("  %-30s %d / %d\n", "Composite Score:", result.CompositeScore, result.MaxScore)
	fmt.Println()

	// --- Recommendation ---------------------------------------------------
	fmt.Println(bold(cyan(divider())))
	recColor := recColorFn(result.Recommendation)
	fmt.Printf("  RECOMMENDATION:  %s\n", bold(recColor(string(result.Recommendation))))
	fmt.Println()

	if len(result.Reasons) > 0 {
		fmt.Println(bold("  Supporting Reasons:"))
		for _, r := range result.Reasons {
			fmt.Printf("  • %s\n", r)
		}
	}
	fmt.Println(bold(cyan(divider())))
	fmt.Println()
}

func printMetric(label, value string) {
	fmt.Printf("  %-30s %s\n", label, value)
}

func pct(f float64) string { return fmt.Sprintf("%.2f%%", f*100) }

func signalStr(s models.ModelSignal) string {
	switch s {
	case models.SignalBuy:
		return green("BUY  (+1)")
	case models.SignalSell:
		return red("SELL (-1)")
	default:
		return yellow("HOLD ( 0)")
	}
}

func recColorFn(r models.Recommendation) func(string) string {
	switch r {
	case models.Buy:
		return green
	case models.Sell:
		return red
	default:
		return yellow
	}
}
