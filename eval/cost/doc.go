// Package cost provides cost-aware evaluation capabilities for comparing
// model configurations on a quality-per-dollar basis.
//
// It includes a CostMetric that computes quality-per-dollar, a ParetoAnalyzer
// that identifies Pareto-optimal configurations, a BudgetAlert that fires
// when evaluation cost exceeds a threshold, and a CostReport with optimization
// recommendations.
//
// Key types:
//   - CostMetric: Computes quality-per-dollar from quality score and token cost
//   - ParetoAnalyzer: Identifies Pareto-optimal model configurations
//   - BudgetAlert: Fires when cumulative cost exceeds a threshold
//   - CostReport: Aggregated analysis with optimization recommendations
package cost
