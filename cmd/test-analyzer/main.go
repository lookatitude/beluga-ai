package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/ast"
	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/cli"
	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/code"
	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/mocks"
	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/patterns"
	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/validation"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Parse flags and run analysis
	config, err := cli.ParseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(4)
	}

	// Show help if requested
	if config.Help {
		cli.PrintHelp()
		os.Exit(0)
	}

	// Validate flags
	if err := cli.ValidateFlags(config); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid flags: %v\n", err)
		os.Exit(4)
	}

	// Initialize components
	parser := ast.NewParser()
	detector := patterns.NewDetector()
	analyzer := NewAnalyzer(detector, parser)

	mockGenerator := mocks.NewGenerator()
	codeModifier := code.NewModifier()
	validator := validation.NewValidator()
	fixer := NewFixer(mockGenerator, codeModifier, validator)

	reporter := NewReporter()

	// Create adapters to bridge type differences
	analyzerAdapter := &analyzerAdapter{analyzer: analyzer}
	fixerAdapter := &fixerAdapter{fixer: fixer}
	reporterAdapter := &reporterAdapter{reporter: reporter}

	// Run analysis
	exitCode, err := cli.RunAnalysis(ctx, config, analyzerAdapter, fixerAdapter, reporterAdapter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running analysis: %v\n", err)
		if exitCode == 0 {
			exitCode = 1
		}
	}

	// Give a moment for cleanup
	time.Sleep(100 * time.Millisecond)
	os.Exit(exitCode)
}

// Adapters to bridge main package types to cli package types
type analyzerAdapter struct {
	analyzer Analyzer
}

func (a *analyzerAdapter) AnalyzePackage(ctx context.Context, packagePath string) (*cli.PackageAnalysis, error) {
	result, err := a.analyzer.AnalyzePackage(ctx, packagePath)
	if err != nil {
		return nil, err
	}
	converted := convertPackageAnalysis(result)
	return &converted, nil
}

type fixerAdapter struct {
	fixer Fixer
}

func (f *fixerAdapter) ApplyFix(ctx context.Context, issue *cli.PerformanceIssue) (*cli.Fix, error) {
	mainIssue := convertPerformanceIssueFromCLI(*issue)
	_, err := f.fixer.ApplyFix(ctx, &mainIssue)
	if err != nil {
		return nil, err
	}
	return &cli.Fix{}, nil // Simplified - would convert fix here
}

func (f *fixerAdapter) ValidateFix(ctx context.Context, fix *cli.Fix) (*cli.ValidationResult, error) {
	// Placeholder
	return &cli.ValidationResult{}, nil
}

func (f *fixerAdapter) RollbackFix(ctx context.Context, fix *cli.Fix) error {
	// Placeholder
	return nil
}

type reporterAdapter struct {
	reporter Reporter
}

func (r *reporterAdapter) GenerateReport(ctx context.Context, report *cli.AnalysisReport, format cli.ReportFormat) ([]byte, error) {
	mainReport := convertAnalysisReport(*report)
	mainFormat := convertReportFormat(format)
	return r.reporter.GenerateReport(ctx, &mainReport, mainFormat)
}

// Conversion functions
func convertPackageAnalysis(pa *PackageAnalysis) cli.PackageAnalysis {
	return cli.PackageAnalysis{
		Package: pa.Package,
		Files:   convertFileAnalyses(pa.Files),
		Issues:  convertPerformanceIssues(pa.Issues),
	}
}

func convertFileAnalyses(files []*FileAnalysis) []*cli.FileAnalysis {
	result := make([]*cli.FileAnalysis, len(files))
	for i, f := range files {
		result[i] = &cli.FileAnalysis{
			Functions: convertTestFunctions(f.Functions),
		}
	}
	return result
}

func convertTestFunctions(funcs []*TestFunction) []*cli.TestFunction {
	result := make([]*cli.TestFunction, len(funcs))
	for i, f := range funcs {
		result[i] = &cli.TestFunction{Name: f.Name}
	}
	return result
}

func convertPerformanceIssues(issues []PerformanceIssue) []cli.PerformanceIssue {
	result := make([]cli.PerformanceIssue, len(issues))
	for i, issue := range issues {
		result[i] = cli.PerformanceIssue{
			Type:     cli.IssueType(issue.Type),
			Severity: cli.Severity(issue.Severity),
		}
	}
	return result
}

func convertPerformanceIssueFromCLI(issue cli.PerformanceIssue) PerformanceIssue {
	return PerformanceIssue{
		Type:     IssueType(issue.Type),
		Severity: Severity(issue.Severity),
	}
}

func convertAnalysisReport(ar cli.AnalysisReport) AnalysisReport {
	return AnalysisReport{
		PackagesAnalyzed:  ar.PackagesAnalyzed,
		FilesAnalyzed:     ar.FilesAnalyzed,
		FunctionsAnalyzed: ar.FunctionsAnalyzed,
		IssuesFound:       ar.IssuesFound,
		IssuesByType:      convertIssuesByTypeFromCLI(ar.IssuesByType),
		IssuesBySeverity:  convertIssuesBySeverityFromCLI(ar.IssuesBySeverity),
		IssuesByPackage:   ar.IssuesByPackage,
		FixesApplied:      ar.FixesApplied,
		FixesFailed:       ar.FixesFailed,
		ExecutionTime:     ar.ExecutionTime,
		GeneratedAt:       ar.GeneratedAt,
	}
}

func convertIssuesByTypeFromCLI(m map[cli.IssueType]int) map[IssueType]int {
	result := make(map[IssueType]int)
	for k, v := range m {
		result[IssueType(k)] = v
	}
	return result
}

func convertIssuesBySeverityFromCLI(m map[cli.Severity]int) map[Severity]int {
	result := make(map[Severity]int)
	for k, v := range m {
		result[Severity(k)] = v
	}
	return result
}

func convertReportFormat(f cli.ReportFormat) ReportFormat {
	return ReportFormat(f)
}
