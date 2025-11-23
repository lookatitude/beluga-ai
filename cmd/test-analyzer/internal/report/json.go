package report

import (
	"context"
	"encoding/json"
	"fmt"
)

// convertIssuesByType converts IssueType map to string map.
func convertIssuesByType(m map[IssueType]int) map[string]int {
	result := make(map[string]int)
	for k, v := range m {
		result[k.String()] = v
	}
	return result
}

// convertIssuesBySeverity converts Severity map to string map.
func convertIssuesBySeverity(m map[Severity]int) map[string]int {
	result := make(map[string]int)
	for k, v := range m {
		result[k.String()] = v
	}
	return result
}

// JSONReportGenerator generates JSON format reports.
type JSONReportGenerator interface {
	GenerateJSONReport(ctx context.Context, report *AnalysisReport) ([]byte, error)
}

// jsonReportGenerator implements JSONReportGenerator.
type jsonReportGenerator struct{}

// NewJSONReportGenerator creates a new JSONReportGenerator.
func NewJSONReportGenerator() JSONReportGenerator {
	return &jsonReportGenerator{}
}

// GenerateJSONReport implements JSONReportGenerator.GenerateJSONReport.
func (g *jsonReportGenerator) GenerateJSONReport(ctx context.Context, report *AnalysisReport) ([]byte, error) {
	// Convert report to JSON-serializable format
	jsonReport := map[string]interface{}{
		"summary": map[string]interface{}{
			"packages_analyzed":  report.PackagesAnalyzed,
			"files_analyzed":     report.FilesAnalyzed,
			"functions_analyzed": report.FunctionsAnalyzed,
			"issues_found":       report.IssuesFound,
			"fixes_applied":      report.FixesApplied,
			"fixes_failed":       report.FixesFailed,
			"execution_time":     report.ExecutionTime.String(),
		},
		"issues_by_type":     convertIssuesByType(report.IssuesByType),
		"issues_by_severity": convertIssuesBySeverity(report.IssuesBySeverity),
		"issues_by_package":  report.IssuesByPackage,
		"generated_at":       report.GeneratedAt.Format("2006-01-02T15:04:05Z"),
	}

	data, err := json.MarshalIndent(jsonReport, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON: %w", err)
	}

	return data, nil
}
