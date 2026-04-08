package cost

import (
	"fmt"
	"strings"
)

// CostReport provides aggregated cost analysis with optimization
// recommendations.
type CostReport struct {
	// TotalCost is the sum of all sample costs in dollars.
	TotalCost float64

	// AverageQuality is the mean quality score across all samples.
	AverageQuality float64

	// QualityPerDollar is the ratio of average quality to total cost.
	QualityPerDollar float64

	// ConfigResults contains per-configuration results used for analysis.
	ConfigResults []ConfigResult

	// ParetoOptimal contains the Pareto-optimal configurations.
	ParetoOptimal []ConfigResult

	// Recommendations contains human-readable optimization suggestions.
	Recommendations []string
}

// GenerateReport creates a CostReport from a set of configuration results.
// It runs Pareto analysis and generates optimization recommendations.
func GenerateReport(configs []ConfigResult) *CostReport {
	if len(configs) == 0 {
		return &CostReport{}
	}

	report := &CostReport{
		ConfigResults: configs,
	}

	// Compute totals.
	var totalCost, totalQuality float64
	for _, c := range configs {
		totalCost += c.Cost
		totalQuality += c.Quality
	}
	report.TotalCost = totalCost
	report.AverageQuality = totalQuality / float64(len(configs))
	if totalCost > 0 {
		report.QualityPerDollar = report.AverageQuality / totalCost
	}

	// Run Pareto analysis.
	analyzer := NewParetoAnalyzer()
	paretoResult := analyzer.Analyze(configs)
	report.ParetoOptimal = paretoResult.Optimal

	// Generate recommendations.
	report.Recommendations = generateRecommendations(configs, paretoResult)

	return report
}

// String returns a formatted text representation of the report.
func (r *CostReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Cost Report\n")
	fmt.Fprintf(&b, "===========\n")
	fmt.Fprintf(&b, "Total Cost:         $%.4f\n", r.TotalCost)
	fmt.Fprintf(&b, "Average Quality:    %.3f\n", r.AverageQuality)
	fmt.Fprintf(&b, "Quality per Dollar: %.2f\n", r.QualityPerDollar)

	if len(r.ParetoOptimal) > 0 {
		b.WriteString("\nPareto-Optimal Configurations:\n")
		for _, c := range r.ParetoOptimal {
			fmt.Fprintf(&b, "  - %s (quality=%.3f, cost=$%.4f)\n", c.Name, c.Quality, c.Cost)
		}
	}

	if len(r.Recommendations) > 0 {
		b.WriteString("\nRecommendations:\n")
		for _, rec := range r.Recommendations {
			fmt.Fprintf(&b, "  - %s\n", rec)
		}
	}

	return b.String()
}

// generateRecommendations produces optimization suggestions from the analysis.
func generateRecommendations(configs []ConfigResult, pareto *ParetoResult) []string {
	var recs []string

	if len(pareto.Dominated) > 0 {
		names := make([]string, len(pareto.Dominated))
		for i, c := range pareto.Dominated {
			names[i] = c.Name
		}
		recs = append(recs, fmt.Sprintf(
			"Consider removing dominated configurations: %s",
			strings.Join(names, ", "),
		))
	}

	// Find best quality-per-dollar among optimal configs.
	var bestQPD float64
	var bestName string
	for _, c := range pareto.Optimal {
		if c.Cost > 0 {
			qpd := c.Quality / c.Cost
			if qpd > bestQPD {
				bestQPD = qpd
				bestName = c.Name
			}
		}
	}
	if bestName != "" {
		recs = append(recs, fmt.Sprintf(
			"Best quality-per-dollar: %s (%.2f quality/$)",
			bestName, bestQPD,
		))
	}

	// Find cheapest option with quality above 0.7.
	var cheapest *ConfigResult
	for i := range configs {
		c := configs[i]
		if c.Quality >= 0.7 {
			if cheapest == nil || c.Cost < cheapest.Cost {
				cheapest = &c
			}
		}
	}
	if cheapest != nil {
		recs = append(recs, fmt.Sprintf(
			"Cheapest option with quality >= 0.7: %s ($%.4f, quality=%.3f)",
			cheapest.Name, cheapest.Cost, cheapest.Quality,
		))
	}

	return recs
}
