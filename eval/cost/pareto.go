package cost

import (
	"sort"
)

// ConfigResult represents a model configuration's evaluation result for
// Pareto analysis.
type ConfigResult struct {
	// Name identifies the configuration (e.g., "gpt-4o", "claude-3-haiku").
	Name string

	// Quality is the average quality score in [0.0, 1.0].
	Quality float64

	// Cost is the total dollar cost of the evaluation run.
	Cost float64
}

// ParetoResult contains the output of a Pareto analysis.
type ParetoResult struct {
	// Optimal contains the Pareto-optimal configurations. A configuration
	// is Pareto-optimal if no other configuration has both higher quality
	// and lower cost.
	Optimal []ConfigResult

	// Dominated contains configurations that are dominated by at least
	// one optimal configuration.
	Dominated []ConfigResult
}

// ParetoAnalyzer identifies Pareto-optimal model configurations from a set
// of quality-cost pairs.
type ParetoAnalyzer struct{}

// NewParetoAnalyzer creates a new ParetoAnalyzer.
func NewParetoAnalyzer() *ParetoAnalyzer {
	return &ParetoAnalyzer{}
}

// Analyze finds the Pareto-optimal configurations. A configuration is
// Pareto-optimal if no other configuration has strictly higher quality and
// strictly lower (or equal) cost, or strictly lower cost and strictly higher
// (or equal) quality.
func (p *ParetoAnalyzer) Analyze(configs []ConfigResult) *ParetoResult {
	if len(configs) == 0 {
		return &ParetoResult{}
	}

	// Sort by cost ascending, then quality descending for stable output.
	sorted := make([]ConfigResult, len(configs))
	copy(sorted, configs)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Cost != sorted[j].Cost {
			return sorted[i].Cost < sorted[j].Cost
		}
		return sorted[i].Quality > sorted[j].Quality
	})

	result := &ParetoResult{}

	for i, c := range sorted {
		if isDominated(sorted, i) {
			result.Dominated = append(result.Dominated, c)
		} else {
			result.Optimal = append(result.Optimal, c)
		}
	}

	return result
}

// isDominated reports whether configs[i] is dominated by any other entry in
// configs: another config with >= quality and <= cost, with at least one
// strict inequality.
func isDominated(configs []ConfigResult, i int) bool {
	c := configs[i]
	for j, other := range configs {
		if i == j {
			continue
		}
		if other.Quality >= c.Quality && other.Cost <= c.Cost &&
			(other.Quality > c.Quality || other.Cost < c.Cost) {
			return true
		}
	}
	return false
}
