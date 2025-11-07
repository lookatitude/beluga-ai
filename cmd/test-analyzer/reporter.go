package main

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/report"
)

// Reporter is the interface for generating analysis reports in various formats.
type Reporter interface {
	// GenerateReport generates a report in the specified format.
	GenerateReport(ctx context.Context, report *AnalysisReport, format ReportFormat) ([]byte, error)

	// GenerateSummary generates a human-readable summary of the analysis.
	GenerateSummary(ctx context.Context, report *AnalysisReport) (string, error)

	// GeneratePackageReport generates a report for a single package.
	GeneratePackageReport(ctx context.Context, packageAnalysis *PackageAnalysis, format ReportFormat) ([]byte, error)
}

// reporter implements the Reporter interface.
type reporter struct {
	jsonGen     report.JSONReportGenerator
	htmlGen     report.HTMLReportGenerator
	markdownGen report.MarkdownReportGenerator
	plainGen    report.PlainReportGenerator
}

// NewReporter creates a new Reporter instance.
func NewReporter() Reporter {
	return &reporter{
		jsonGen:     report.NewJSONReportGenerator(),
		htmlGen:     report.NewHTMLReportGenerator(),
		markdownGen: report.NewMarkdownReportGenerator(),
		plainGen:    report.NewPlainReportGenerator(),
	}
}

// GenerateReport implements Reporter.GenerateReport.
func (r *reporter) GenerateReport(ctx context.Context, reportData *AnalysisReport, format ReportFormat) ([]byte, error) {
	// Convert main package types to report package types
	report := convertToReportAnalysisReport(reportData)
	
	switch format {
	case FormatJSON:
		return r.jsonGen.GenerateJSONReport(ctx, report)
	case FormatHTML:
		return r.htmlGen.GenerateHTMLReport(ctx, report)
	case FormatMarkdown:
		return r.markdownGen.GenerateMarkdownReport(ctx, report)
	case FormatPlain:
		return r.plainGen.GeneratePlainReport(ctx, report)
	default:
		return nil, fmt.Errorf("unsupported format: %v", format)
	}
}

// convertToReportAnalysisReport converts main package AnalysisReport to report package AnalysisReport.
func convertToReportAnalysisReport(r *AnalysisReport) *report.AnalysisReport {
	// This is a simplified conversion - would need full type mapping
	return &report.AnalysisReport{
		PackagesAnalyzed:  r.PackagesAnalyzed,
		FilesAnalyzed:     r.FilesAnalyzed,
		FunctionsAnalyzed: r.FunctionsAnalyzed,
		IssuesFound:       r.IssuesFound,
		IssuesByType:      convertIssuesByTypeToReport(r.IssuesByType),
		IssuesBySeverity:  convertIssuesBySeverityToReport(r.IssuesBySeverity),
		IssuesByPackage:   r.IssuesByPackage,
		FixesApplied:      r.FixesApplied,
		FixesFailed:       r.FixesFailed,
		ExecutionTime:     r.ExecutionTime,
		GeneratedAt:       r.GeneratedAt,
	}
}

// convertIssuesByTypeToReport converts IssueType map.
func convertIssuesByTypeToReport(m map[IssueType]int) map[report.IssueType]int {
	result := make(map[report.IssueType]int)
	for k, v := range m {
		result[report.IssueType(k)] = v
	}
	return result
}

// convertIssuesBySeverityToReport converts Severity map.
func convertIssuesBySeverityToReport(m map[Severity]int) map[report.Severity]int {
	result := make(map[report.Severity]int)
	for k, v := range m {
		result[report.Severity(k)] = v
	}
	return result
}

// GenerateSummary implements Reporter.GenerateSummary.
func (r *reporter) GenerateSummary(ctx context.Context, report *AnalysisReport) (string, error) {
	summary := fmt.Sprintf(
		"Analysis Summary\n"+
			"================\n"+
			"Packages Analyzed: %d\n"+
			"Files Analyzed: %d\n"+
			"Functions Analyzed: %d\n"+
			"Issues Found: %d\n"+
			"Fixes Applied: %d\n"+
			"Fixes Failed: %d\n"+
			"Execution Time: %v\n"+
			"Generated At: %v\n",
		report.PackagesAnalyzed,
		report.FilesAnalyzed,
		report.FunctionsAnalyzed,
		report.IssuesFound,
		report.FixesApplied,
		report.FixesFailed,
		report.ExecutionTime,
		report.GeneratedAt,
	)

	return summary, nil
}

// GeneratePackageReport implements Reporter.GeneratePackageReport.
func (r *reporter) GeneratePackageReport(ctx context.Context, packageAnalysis *PackageAnalysis, format ReportFormat) ([]byte, error) {
	// Convert PackageAnalysis to AnalysisReport format
	report := &AnalysisReport{
		PackagesAnalyzed:  1,
		FilesAnalyzed:     len(packageAnalysis.Files),
		FunctionsAnalyzed: packageAnalysis.Summary.TotalFunctions,
		IssuesFound:        packageAnalysis.Summary.TotalIssues,
		IssuesByType:       packageAnalysis.Summary.IssuesByType,
		IssuesBySeverity:   packageAnalysis.Summary.IssuesBySeverity,
		IssuesByPackage:    map[string]int{packageAnalysis.Package: packageAnalysis.Summary.TotalIssues},
		GeneratedAt:        packageAnalysis.AnalyzedAt,
		Packages:           []*PackageAnalysis{packageAnalysis},
	}

	return r.GenerateReport(ctx, report, format)
}

