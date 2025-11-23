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

	buf.WriteString("# Test Analysis Report\n\n")
	buf.WriteString("## Summary\n\n")
	buf.WriteString(fmt.Sprintf("- Packages Analyzed: %d\n", report.PackagesAnalyzed))
	buf.WriteString(fmt.Sprintf("- Files Analyzed: %d\n", report.FilesAnalyzed))
	buf.WriteString(fmt.Sprintf("- Functions Analyzed: %d\n", report.FunctionsAnalyzed))
	buf.WriteString(fmt.Sprintf("- Issues Found: %d\n", report.IssuesFound))
	buf.WriteString(fmt.Sprintf("- Fixes Applied: %d\n", report.FixesApplied))
	buf.WriteString(fmt.Sprintf("- Fixes Failed: %d\n", report.FixesFailed))
	buf.WriteString(fmt.Sprintf("- Execution Time: %s\n\n", report.ExecutionTime))

	buf.WriteString("## Issues by Type\n\n")
	buf.WriteString("| Type | Count |\n")
	buf.WriteString("|------|-------|\n")
	for issueType, count := range report.IssuesByType {
		buf.WriteString(fmt.Sprintf("| %s | %d |\n", issueType.String(), count))
	}

	buf.WriteString("\n## Issues by Severity\n\n")
	buf.WriteString("| Severity | Count |\n")
	buf.WriteString("|----------|-------|\n")
	for severity, count := range report.IssuesBySeverity {
		buf.WriteString(fmt.Sprintf("| %s | %d |\n", severity.String(), count))
	}

	buf.WriteString(fmt.Sprintf("\n\n*Generated at: %s*\n", report.GeneratedAt.Format("2006-01-02 15:04:05")))

	return []byte(buf.String()), nil
}
