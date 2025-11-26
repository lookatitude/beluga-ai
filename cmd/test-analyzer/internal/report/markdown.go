package report

import (
	"context"
	"fmt"
	"strings"
)

// MarkdownReportGenerator generates Markdown format reports.
type MarkdownReportGenerator interface {
	GenerateMarkdownReport(ctx context.Context, report *AnalysisReport) ([]byte, error)
}

// markdownReportGenerator implements MarkdownReportGenerator.
type markdownReportGenerator struct{}

// NewMarkdownReportGenerator creates a new MarkdownReportGenerator.
func NewMarkdownReportGenerator() MarkdownReportGenerator {
	return &markdownReportGenerator{}
}

// GenerateMarkdownReport implements MarkdownReportGenerator.GenerateMarkdownReport.
func (g *markdownReportGenerator) GenerateMarkdownReport(ctx context.Context, report *AnalysisReport) ([]byte, error) {
	var buf strings.Builder

	_, _ = buf.WriteString("# Test Analysis Report\n\n")
	_, _ = buf.WriteString("## Summary\n\n")
	_, _ = buf.WriteString(fmt.Sprintf("- Packages Analyzed: %d\n", report.PackagesAnalyzed))
	_, _ = buf.WriteString(fmt.Sprintf("- Files Analyzed: %d\n", report.FilesAnalyzed))
	_, _ = buf.WriteString(fmt.Sprintf("- Functions Analyzed: %d\n", report.FunctionsAnalyzed))
	_, _ = buf.WriteString(fmt.Sprintf("- Issues Found: %d\n", report.IssuesFound))
	_, _ = buf.WriteString(fmt.Sprintf("- Fixes Applied: %d\n", report.FixesApplied))
	_, _ = buf.WriteString(fmt.Sprintf("- Fixes Failed: %d\n", report.FixesFailed))
	_, _ = buf.WriteString(fmt.Sprintf("- Execution Time: %s\n\n", report.ExecutionTime))

	_, _ = buf.WriteString("## Issues by Type\n\n")
	_, _ = buf.WriteString("| Type | Count |\n")
	_, _ = buf.WriteString("|------|-------|\n")
	for issueType, count := range report.IssuesByType {
		_, _ = buf.WriteString(fmt.Sprintf("| %s | %d |\n", issueType.String(), count))
	}

	_, _ = buf.WriteString("\n## Issues by Severity\n\n")
	_, _ = buf.WriteString("| Severity | Count |\n")
	_, _ = buf.WriteString("|----------|-------|\n")
	for severity, count := range report.IssuesBySeverity {
		_, _ = buf.WriteString(fmt.Sprintf("| %s | %d |\n", severity.String(), count))
	}

	_, _ = buf.WriteString(fmt.Sprintf("\n\n*Generated at: %s*\n", report.GeneratedAt.Format("2006-01-02 15:04:05")))

	return []byte(buf.String()), nil
}
