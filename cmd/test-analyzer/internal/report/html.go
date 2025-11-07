package report

import (
	"context"
	"fmt"
	"html/template"
	"strings"
)

// HTMLReportGenerator generates HTML format reports.
type HTMLReportGenerator interface {
	GenerateHTMLReport(ctx context.Context, report *AnalysisReport) ([]byte, error)
}

// htmlReportGenerator implements HTMLReportGenerator.
type htmlReportGenerator struct{}

// NewHTMLReportGenerator creates a new HTMLReportGenerator.
func NewHTMLReportGenerator() HTMLReportGenerator {
	return &htmlReportGenerator{}
}

// GenerateHTMLReport implements HTMLReportGenerator.GenerateHTMLReport.
func (g *htmlReportGenerator) GenerateHTMLReport(ctx context.Context, report *AnalysisReport) ([]byte, error) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
	<title>Test Analysis Report</title>
	<style>
		body { font-family: Arial, sans-serif; margin: 20px; }
		.summary { background: #f5f5f5; padding: 20px; border-radius: 5px; }
		.issue { margin: 10px 0; padding: 10px; border-left: 4px solid #ccc; }
		.critical { border-color: #d32f2f; }
		.high { border-color: #f57c00; }
		.medium { border-color: #fbc02d; }
		.low { border-color: #388e3c; }
		table { border-collapse: collapse; width: 100%; margin: 20px 0; }
		th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
		th { background-color: #4CAF50; color: white; }
	</style>
</head>
<body>
	<h1>Test Analysis Report</h1>
	<div class="summary">
		<h2>Summary</h2>
		<p>Packages Analyzed: {{.PackagesAnalyzed}}</p>
		<p>Files Analyzed: {{.FilesAnalyzed}}</p>
		<p>Functions Analyzed: {{.FunctionsAnalyzed}}</p>
		<p>Issues Found: {{.IssuesFound}}</p>
		<p>Fixes Applied: {{.FixesApplied}}</p>
		<p>Fixes Failed: {{.FixesFailed}}</p>
		<p>Execution Time: {{.ExecutionTime}}</p>
	</div>
	<h2>Issues by Type</h2>
	<table>
		<tr><th>Type</th><th>Count</th></tr>
		{{range $type, $count := .IssuesByType}}
		<tr><td>{{$type}}</td><td>{{$count}}</td></tr>
		{{end}}
	</table>
	<h2>Issues by Severity</h2>
	<table>
		<tr><th>Severity</th><th>Count</th></tr>
		{{range $severity, $count := .IssuesBySeverity}}
		<tr><td>{{$severity}}</td><td>{{$count}}</td></tr>
		{{end}}
	</table>
</body>
</html>`

	data := struct {
		PackagesAnalyzed  int
		FilesAnalyzed     int
		FunctionsAnalyzed  int
		IssuesFound       int
		FixesApplied      int
		FixesFailed       int
		ExecutionTime     string
		IssuesByType      map[string]int
		IssuesBySeverity  map[string]int
	}{
		PackagesAnalyzed:  report.PackagesAnalyzed,
		FilesAnalyzed:     report.FilesAnalyzed,
		FunctionsAnalyzed: report.FunctionsAnalyzed,
		IssuesFound:       report.IssuesFound,
		FixesApplied:      report.FixesApplied,
		FixesFailed:       report.FixesFailed,
		ExecutionTime:     report.ExecutionTime.String(),
		IssuesByType:      convertIssuesByType(report.IssuesByType),
		IssuesBySeverity:  convertIssuesBySeverityForHTML(report.IssuesBySeverity),
	}

	t, err := template.New("html").Parse(tmpl)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return []byte(buf.String()), nil
}

// convertIssuesByTypeForHTML converts IssueType map to string map.
func convertIssuesByTypeForHTML(m map[IssueType]int) map[string]int {
	result := make(map[string]int)
	for k, v := range m {
		result[k.String()] = v
	}
	return result
}

// convertIssuesBySeverityForHTML converts Severity map to string map.
func convertIssuesBySeverityForHTML(m map[Severity]int) map[string]int {
	result := make(map[string]int)
	for k, v := range m {
		result[k.String()] = v
	}
	return result
}

