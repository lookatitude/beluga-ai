package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Analyzer, Fixer, Reporter are interfaces that will be implemented in main package
// These are defined here to avoid circular dependencies
type Analyzer interface {
	AnalyzePackage(ctx context.Context, packagePath string) (*PackageAnalysis, error)
}

type Fixer interface {
	ApplyFix(ctx context.Context, issue *PerformanceIssue) (*Fix, error)
	ValidateFix(ctx context.Context, fix *Fix) (*ValidationResult, error)
	RollbackFix(ctx context.Context, fix *Fix) error
}

type Reporter interface {
	GenerateReport(ctx context.Context, report *AnalysisReport, format ReportFormat) ([]byte, error)
}

// Types used by runner (simplified versions)
type PackageAnalysis struct {
	Package string
	Files   []*FileAnalysis
	Issues  []PerformanceIssue
}

type FileAnalysis struct {
	Functions []*TestFunction
}

type TestFunction struct {
	Name string
}

type PerformanceIssue struct {
	Type        IssueType
	Severity    Severity
	Location    Location
	Description string
	Context     map[string]interface{}
	Fixable     bool
}

type Location struct {
	Package     string
	File        string
	Function    string
	LineStart   int
	LineEnd     int
	ColumnStart int
	ColumnEnd   int
}

type IssueType int

func (t IssueType) String() string { return "IssueType" }

type Severity int

func (s Severity) String() string { return "Severity" }

type Fix struct{}
type ValidationResult struct{}

type AnalysisReport struct {
	PackagesAnalyzed  int
	FilesAnalyzed     int
	FunctionsAnalyzed int
	IssuesFound       int
	IssuesByType      map[IssueType]int
	IssuesBySeverity  map[Severity]int
	IssuesByPackage   map[string]int
	FixesApplied      int
	FixesFailed       int
	ExecutionTime     time.Duration
	GeneratedAt       time.Time
	Packages          []PackageAnalysis
}

type ReportFormat int

const (
	FormatJSON ReportFormat = iota
	FormatHTML
	FormatMarkdown
	FormatPlain
)

// RunAnalysis runs the analysis based on the configuration.
// analyzer, fixer, and reporter should be passed from main package.
func RunAnalysis(ctx context.Context, config *Config, analyzer Analyzer, fixer Fixer, reporter Reporter) (int, error) {
	startTime := time.Now()

	// Determine packages to analyze
	packages := config.PackagePaths
	if len(packages) == 0 {
		// Default: analyze all packages in pkg/
		packages = findPackages("pkg")
	}

	// Filter excluded packages
	packages = filterPackages(packages, config.ExcludePackages)

	// Analyze packages
	var allPackages []PackageAnalysis
	totalIssues := 0
	totalFixes := 0
	totalFailedFixes := 0

	for _, pkg := range packages {
		pkgAnalysis, err := analyzer.AnalyzePackage(ctx, pkg)
		if err != nil {
			if !config.Quiet {
				fmt.Fprintf(os.Stderr, "Error analyzing package %s: %v\n", pkg, err)
			}
			continue
		}

		allPackages = append(allPackages, *pkgAnalysis)
		totalIssues += len(pkgAnalysis.Issues)

		// Apply fixes if auto-fix is enabled
		if config.AutoFix && !config.DryRun {
			for i := range pkgAnalysis.Issues {
				issue := &pkgAnalysis.Issues[i]
				if shouldFixIssue(*issue, config) {
					fix, err := fixer.ApplyFix(ctx, issue)
					if err != nil {
						if !config.Quiet {
							fmt.Fprintf(os.Stderr, "Failed to apply fix for issue in %s: %v\n",
								pkg, err)
						}
						totalFailedFixes++
						continue
					}

					// Validate fix
					if !config.SkipValidation {
						_, err := fixer.ValidateFix(ctx, fix)
						if err != nil {
							if !config.Quiet {
								fmt.Fprintf(os.Stderr, "Failed to validate fix in %s: %v\n",
									pkg, err)
							}
							_ = fixer.RollbackFix(ctx, fix)
							totalFailedFixes++
							continue
						}
					}

					totalFixes++
				}
			}
		}
	}

	// Generate report
	report := &AnalysisReport{
		PackagesAnalyzed:  len(allPackages),
		FilesAnalyzed:     countFiles(allPackages),
		FunctionsAnalyzed: countFunctions(allPackages),
		IssuesFound:       totalIssues,
		IssuesByType:      aggregateIssuesByType(allPackages),
		IssuesBySeverity:  aggregateIssuesBySeverity(allPackages),
		IssuesByPackage:   aggregateIssuesByPackage(allPackages),
		FixesApplied:      totalFixes,
		FixesFailed:       totalFailedFixes,
		ExecutionTime:     time.Since(startTime),
		GeneratedAt:       time.Now(),
		Packages:          allPackages,
	}

	// Output report
	format := parseFormat(config.Output)
	reportData, err := reporter.GenerateReport(ctx, report, format)
	if err != nil {
		return 1, fmt.Errorf("generating report: %w", err)
	}

	// Write output
	if config.OutputFile != "" {
		if err := os.WriteFile(config.OutputFile, reportData, 0644); err != nil {
			return 1, fmt.Errorf("writing output file: %w", err)
		}
	} else {
		os.Stdout.Write(reportData)
	}

	// Determine exit code
	if totalFailedFixes > 0 {
		return 2, nil // Fix errors
	}
	if totalIssues > 0 && config.DryRun {
		return 0, nil // Issues found but dry-run
	}
	return 0, nil
}

// Helper functions
func findPackages(root string) []string {
	var packages []string
	seen := make(map[string]bool)

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// Find test files and extract their package directory
		if !info.IsDir() && strings.HasSuffix(path, "_test.go") {
			pkgDir := filepath.Dir(path)
			// Avoid duplicates
			if !seen[pkgDir] {
				packages = append(packages, pkgDir)
				seen[pkgDir] = true
			}
		}
		return nil
	})
	return packages
}

func filterPackages(packages, exclude []string) []string {
	if len(exclude) == 0 {
		return packages
	}

	var filtered []string
	for _, pkg := range packages {
		excluded := false
		for _, excl := range exclude {
			if strings.Contains(pkg, excl) {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, pkg)
		}
	}
	return filtered
}

func shouldFixIssue(issue PerformanceIssue, config *Config) bool {
	// Check if fix type is in allowed list
	if len(config.FixTypes) > 0 {
		found := false
		for _, fixType := range config.FixTypes {
			if strings.EqualFold(fixType, issue.Type.String()) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func parseFormat(output string) ReportFormat {
	switch strings.ToLower(output) {
	case "json":
		return FormatJSON
	case "html":
		return FormatHTML
	case "markdown":
		return FormatMarkdown
	case "plain":
		return FormatPlain
	default:
		return FormatPlain
	}
}

func countFiles(packages []PackageAnalysis) int {
	total := 0
	for _, pkg := range packages {
		total += len(pkg.Files)
	}
	return total
}

func countFunctions(packages []PackageAnalysis) int {
	total := 0
	for _, pkg := range packages {
		for _, file := range pkg.Files {
			if file != nil {
				total += len(file.Functions)
			}
		}
	}
	return total
}

func aggregateIssuesByType(packages []PackageAnalysis) map[IssueType]int {
	result := make(map[IssueType]int)
	for _, pkg := range packages {
		for _, issue := range pkg.Issues {
			result[issue.Type]++
		}
	}
	return result
}

func aggregateIssuesBySeverity(packages []PackageAnalysis) map[Severity]int {
	result := make(map[Severity]int)
	for _, pkg := range packages {
		for _, issue := range pkg.Issues {
			result[issue.Severity]++
		}
	}
	return result
}

func aggregateIssuesByPackage(packages []PackageAnalysis) map[string]int {
	result := make(map[string]int)
	for _, pkg := range packages {
		result[pkg.Package] = len(pkg.Issues)
	}
	return result
}
