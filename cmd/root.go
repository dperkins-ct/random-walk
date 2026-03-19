// Package cmd wires together the CLI for the random-walk stock analyzer.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dperkins-ct/random-walk/internal/analysis"
	"github.com/dperkins-ct/random-walk/internal/api"
	"github.com/dperkins-ct/random-walk/internal/recommendation"
)

var (
	apiKey  string
	verbose bool
)

// rootCmd is the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "random-walk",
	Short: "random-walk – CLI stock analyzer",
	Long: `random-walk fetches stock price data, runs technical analysis
(SMA, RSI, MACD), and outputs a BUY / SELL / HOLD recommendation.`,
}

// Execute adds all child commands and runs rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "",
		"Alpha Vantage API key (or set ALPHAVANTAGE_API_KEY env var)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"print detailed indicator values alongside the recommendation")

	rootCmd.AddCommand(analyzeCmd)
}

// analyzeCmd is the "analyze" subcommand.
var analyzeCmd = &cobra.Command{
	Use:   "analyze <TICKER> [TICKER...]",
	Short: "Analyze one or more stock tickers",
	Long: `Fetch daily price data for each ticker, compute technical indicators,
and print a BUY / SELL / HOLD recommendation.

Example:
  random-walk analyze AAPL MSFT GOOG --api-key YOUR_KEY
  ALPHAVANTAGE_API_KEY=YOUR_KEY random-walk analyze TSLA -v`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAnalyze,
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	key := resolveAPIKey()
	if key == "" {
		return fmt.Errorf(
			"Alpha Vantage API key is required.\n" +
				"  Use --api-key <key>  or  set ALPHAVANTAGE_API_KEY environment variable.\n" +
				"  Get a free key at https://www.alphavantage.co/support/#api-key",
		)
	}

	client := api.NewClient(key)

	for _, ticker := range args {
		ticker = strings.ToUpper(ticker)
		fmt.Printf("\n=== %s ===\n", ticker)

		prices, err := client.GetDailyPrices(ticker)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%s] error fetching data: %v\n", ticker, err)
			continue
		}

		closes := api.ClosePrices(prices)
		result, err := analysis.Analyze(closes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%s] error analyzing data: %v\n", ticker, err)
			continue
		}

		rec := recommendation.Generate(ticker, result)

		fmt.Printf("Recommendation : %s  (score: %+d)\n", rec.Action, rec.Score)
		fmt.Printf("Latest Close   : $%.2f\n", result.LatestClose)

		if verbose {
			fmt.Printf("SMA20          : $%.2f\n", result.SMA20)
			if result.SMA50 > 0 {
				fmt.Printf("SMA50          : $%.2f\n", result.SMA50)
			}
			fmt.Printf("RSI14          : %.2f\n", result.RSI14)
			fmt.Printf("MACD           : %.4f\n", result.MACD)
			fmt.Printf("MACD Signal    : %.4f\n", result.MACDSignal)
			fmt.Printf("MACD Histogram : %.4f\n", result.MACDHistogram)
		}

		fmt.Println("Signals:")
		for _, reason := range rec.Reasons {
			fmt.Printf("  • %s\n", reason)
		}
	}

	return nil
}

// resolveAPIKey returns the API key from the flag, falling back to the
// ALPHAVANTAGE_API_KEY environment variable.
func resolveAPIKey() string {
	if apiKey != "" {
		return apiKey
	}
	return os.Getenv("ALPHAVANTAGE_API_KEY")
}
