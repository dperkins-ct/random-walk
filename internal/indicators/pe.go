package indicators

// cheapThreshold: P/E below this is considered undervalued vs S&P 500 median (~25).
const cheapThreshold = 18.0

// expensiveThreshold: P/E above this is considered overvalued.
const expensiveThreshold = 30.0

// EvaluatePE classifies a P/E ratio relative to S&P 500 historical median.
// A zero or negative P/E (company has no earnings) returns Fair (neutral).
func EvaluatePE(pe float64) PESignal {
	if pe <= 0 {
		return Fair
	}
	if pe < cheapThreshold {
		return Cheap
	}
	if pe > expensiveThreshold {
		return Expensive
	}
	return Fair
}
