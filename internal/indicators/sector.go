package indicators

// SectorETFMap maps Alpha Vantage sector names to their most-liquid SPDR ETF.
// If a sector is not found, the caller should skip sector-ETF comparison.
var SectorETFMap = map[string]string{
	"Technology":             "XLK",
	"Health Care":            "XLV",
	"Financials":             "XLF",
	"Consumer Discretionary": "XLY",
	"Consumer Staples":       "XLP",
	"Energy":                 "XLE",
	"Utilities":              "XLU",
	"Industrials":            "XLI",
	"Materials":              "XLB",
	"Real Estate":            "XLRE",
	"Communication Services": "XLC",
}

// SectorETF returns the ETF ticker for a given sector name, and whether it
// was found in the map.
func SectorETF(sector string) (string, bool) {
	etf, ok := SectorETFMap[sector]
	return etf, ok
}
