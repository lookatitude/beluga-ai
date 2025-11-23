package report

import (
	"context"
	"fmt"
	"strings"
)

// PlainReportGenerator generates plain text format reports.
type PlainReportGenerator interface {
	GeneratePlainReport(ctx context.Context, report *AnalysisReport) ([]byte, error)
}

// plainReportGenerator implements PlainReportGenerator.
type plainReportGenerator struct{}

// NewPlainReportGenerator creates a new PlainReportGenerator.
func NewPlainReportGenerator() PlainReportGenerator {
	return &plainReportGenerator{}
}

// GeneratePlainReport implements PlainReportGenerator.GeneratePlainReport.
func (g *plainReportGenerator) GeneratePlainReport(ctx context.Context, report *AnalysisReport) ([]byte, error) {
	var buf strings.Builder

	buf.WriteString("Test Analysis Report\n")
	buf.WriteString(strings.Repeat("=", 50) + "\n\n")
	buf.WriteString("Summary:\n")
	buf.WriteString(fmt.Sprintf("  Packages Analyzed: %d\n", report.PackagesAnalyzed))
	buf.WriteString(fmt.Sprintf("  Files Analyzed: %d\n", report.FilesAnalyzed))
	buf.WriteString(fmt.Sprintf("  Functions Analyzed: %d\n", report.FunctionsAnalyzed))
	buf.WriteString(fmt.Sprintf("  Issues Found: %d\n", report.IssuesFound))
	buf.WriteString(fmt.Sprintf("  Fixes Applied: %d\n", report.FixesApplied))
	buf.WriteString(fmt.Sprintf("  Fixes Failed: %d\n", report.FixesFailed))
	buf.WriteString(fmt.Sprintf("  Execution Time: %s\n\n", report.ExecutionTime))

	buf.WriteString("Issues by Type:\n")
	for issueType, count := range report.IssuesByType {
		buf.WriteString(fmt.Sprintf("  %s: %d\n", issueType.String(), count))
	}

	buf.WriteString("\nIssues by Severity:\n")
	for severity, count := range report.IssuesBySeverity {
		buf.WriteString(fmt.Sprintf("  %s: %d\n", severity.String(), count))
	}

	buf.WriteString(fmt.Sprintf("\nGenerated at: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05")))

	return []byte(buf.String()), nil
}
