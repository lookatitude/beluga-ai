package report

import (
	"time"
)

// AnalysisReport represents the aggregated results of the analysis.
// This is a copy of the main package types for the report package.
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
	Packages          []*PackageAnalysis
}

// PackageAnalysis represents analysis results for a package.
type PackageAnalysis struct {
	Package    string
	Files      []*FileAnalysis
	Issues     []PerformanceIssue
	Summary    *AnalysisSummary
	AnalyzedAt time.Time
}

// FileAnalysis represents analysis results for a file.
type FileAnalysis struct {
	File       *TestFile
	Functions  []*TestFunction
	Issues     []PerformanceIssue
	AnalyzedAt time.Time
}

// TestFile represents a test file.
type TestFile struct {
	Path string
}

// TestFunction represents a test function.
type TestFunction struct {
	Name string
}

// PerformanceIssue represents a performance issue.
type PerformanceIssue struct {
	Type string
}

// AnalysisSummary represents summary statistics.
type AnalysisSummary struct {
	TotalFiles       int
	TotalFunctions   int
	TotalIssues      int
	IssuesByType     map[IssueType]int
	IssuesBySeverity map[Severity]int
}

// IssueType represents an issue type.
type IssueType int

const (
	ReportIssueTypeInfiniteLoop IssueType = iota
	ReportIssueTypeMissingTimeout
	ReportIssueTypeLargeIteration
	ReportIssueTypeHighConcurrency
	ReportIssueTypeSleepDelay
	ReportIssueTypeActualImplementationUsage
	ReportIssueTypeMixedMockRealUsage
	ReportIssueTypeMissingMock
	ReportIssueTypeBenchmarkHelperUsage
	ReportIssueTypeOther
)

func (t IssueType) String() string {
	switch t {
	case ReportIssueTypeInfiniteLoop:
		return "InfiniteLoop"
	case ReportIssueTypeMissingTimeout:
		return "MissingTimeout"
	case ReportIssueTypeLargeIteration:
		return "LargeIteration"
	case ReportIssueTypeHighConcurrency:
		return "HighConcurrency"
	case ReportIssueTypeSleepDelay:
		return "SleepDelay"
	case ReportIssueTypeActualImplementationUsage:
		return "ActualImplementationUsage"
	case ReportIssueTypeMixedMockRealUsage:
		return "MixedMockRealUsage"
	case ReportIssueTypeMissingMock:
		return "MissingMock"
	case ReportIssueTypeBenchmarkHelperUsage:
		return "BenchmarkHelperUsage"
	case ReportIssueTypeOther:
		return "Other"
	default:
		return "Unknown"
	}
}

// Severity represents issue severity.
type Severity int

const (
	ReportSeverityLow Severity = iota
	ReportSeverityMedium
	ReportSeverityHigh
	ReportSeverityCritical
)

func (s Severity) String() string {
	switch s {
	case ReportSeverityLow:
		return "Low"
	case ReportSeverityMedium:
		return "Medium"
	case ReportSeverityHigh:
		return "High"
	case ReportSeverityCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}
