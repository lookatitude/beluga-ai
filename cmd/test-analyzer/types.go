package main

import (
	"go/ast"
	"time"
)

// TestFile represents a Go test file containing test functions and test utilities.
type TestFile struct {
	Path                 string
	Package              string
	Functions            []*TestFunction
	HasIntegrationSuffix bool
	AST                  *ast.File
}

// TestFunction represents an individual test function that may contain long-running operations.
type TestFunction struct {
	Name                     string
	Type                     TestType
	File                     *TestFile
	LineStart                int
	LineEnd                  int
	Issues                   []PerformanceIssue
	HasTimeout               bool
	TimeoutDuration          time.Duration
	ExecutionTime            time.Duration
	UsesActualImplementation bool
	UsesMocks                bool
	MixedUsage               bool
}

// TestType is an enumeration of test function types.
type TestType int

const (
	TestTypeUnit TestType = iota
	TestTypeIntegration
	TestTypeLoad
)

// String returns the string representation of TestType.
func (t TestType) String() string {
	switch t {
	case TestTypeUnit:
		return "Unit"
	case TestTypeIntegration:
		return "Integration"
	case TestTypeLoad:
		return "Load"
	default:
		return "Unknown"
	}
}

// PerformanceIssue represents an identified problem in a test function.
type PerformanceIssue struct {
	Type        IssueType
	Severity    Severity
	Location    Location
	Description string
	Context     map[string]interface{}
	Fixable     bool
	Fix         *Fix
}

// IssueType is an enumeration of performance issue categories.
type IssueType int

const (
	IssueTypeInfiniteLoop IssueType = iota
	IssueTypeMissingTimeout
	IssueTypeLargeIteration
	IssueTypeHighConcurrency
	IssueTypeSleepDelay
	IssueTypeActualImplementationUsage
	IssueTypeMixedMockRealUsage
	IssueTypeMissingMock
	IssueTypeBenchmarkHelperUsage
	IssueTypeOther
)

// String returns the string representation of IssueType.
func (t IssueType) String() string {
	switch t {
	case IssueTypeInfiniteLoop:
		return "InfiniteLoop"
	case IssueTypeMissingTimeout:
		return "MissingTimeout"
	case IssueTypeLargeIteration:
		return "LargeIteration"
	case IssueTypeHighConcurrency:
		return "HighConcurrency"
	case IssueTypeSleepDelay:
		return "SleepDelay"
	case IssueTypeActualImplementationUsage:
		return "ActualImplementationUsage"
	case IssueTypeMixedMockRealUsage:
		return "MixedMockRealUsage"
	case IssueTypeMissingMock:
		return "MissingMock"
	case IssueTypeBenchmarkHelperUsage:
		return "BenchmarkHelperUsage"
	case IssueTypeOther:
		return "Other"
	default:
		return "Unknown"
	}
}

// Severity is an enumeration of issue severity levels.
type Severity int

const (
	SeverityLow Severity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// String returns the string representation of Severity.
func (s Severity) String() string {
	switch s {
	case SeverityLow:
		return "Low"
	case SeverityMedium:
		return "Medium"
	case SeverityHigh:
		return "High"
	case SeverityCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

// AssignSeverity assigns severity based on issue type and test type.
func AssignSeverity(issueType IssueType, testType TestType) Severity {
	switch issueType {
	case IssueTypeInfiniteLoop:
		return SeverityCritical
	case IssueTypeMissingTimeout:
		if testType == TestTypeUnit {
			return SeverityHigh
		}
		return SeverityMedium
	case IssueTypeLargeIteration:
		return SeverityMedium // Will be adjusted based on operation complexity
	case IssueTypeActualImplementationUsage:
		if testType == TestTypeUnit {
			return SeverityHigh
		}
		return SeverityLow
	case IssueTypeMixedMockRealUsage:
		if testType == TestTypeUnit {
			return SeverityMedium
		}
		return SeverityLow
	case IssueTypeMissingMock:
		return SeverityMedium
	case IssueTypeSleepDelay:
		return SeverityLow // Will be adjusted based on duration
	default:
		return SeverityMedium
	}
}

// Location represents the location of an issue in source code.
type Location struct {
	Package     string
	File        string
	Function    string
	LineStart   int
	LineEnd     int
	ColumnStart int
	ColumnEnd   int
}

// Fix represents a proposed or applied fix for a performance issue.
type Fix struct {
	Issue            *PerformanceIssue
	Type             FixType
	Changes          []CodeChange
	Status           FixStatus
	ValidationResult *ValidationResult
	BackupPath       string
	AppliedAt        time.Time
}

// FixType is an enumeration of fix types.
type FixType int

const (
	FixTypeUnknown FixType = iota
	FixTypeAddTimeout
	FixTypeReduceIterations
	FixTypeOptimizeSleep
	FixTypeAddLoopExit
	FixTypeReplaceWithMock
	FixTypeCreateMock
	FixTypeUpdateTestFile
)

// String returns the string representation of FixType.
func (f FixType) String() string {
	switch f {
	case FixTypeAddTimeout:
		return "AddTimeout"
	case FixTypeReduceIterations:
		return "ReduceIterations"
	case FixTypeOptimizeSleep:
		return "OptimizeSleep"
	case FixTypeAddLoopExit:
		return "AddLoopExit"
	case FixTypeReplaceWithMock:
		return "ReplaceWithMock"
	case FixTypeCreateMock:
		return "CreateMock"
	case FixTypeUpdateTestFile:
		return "UpdateTestFile"
	default:
		return "Unknown"
	}
}

// FixStatus is an enumeration of fix statuses.
type FixStatus int

const (
	FixStatusProposed FixStatus = iota
	FixStatusApplied
	FixStatusValidated
	FixStatusFailed
	FixStatusRolledBack
)

// String returns the string representation of FixStatus.
func (s FixStatus) String() string {
	switch s {
	case FixStatusProposed:
		return "Proposed"
	case FixStatusApplied:
		return "Applied"
	case FixStatusValidated:
		return "Validated"
	case FixStatusFailed:
		return "Failed"
	case FixStatusRolledBack:
		return "RolledBack"
	default:
		return "Unknown"
	}
}

// CodeChange represents a single code modification.
type CodeChange struct {
	File        string
	LineStart   int
	LineEnd     int
	OldCode     string
	NewCode     string
	Description string
}

// ValidationResult represents the result of fix validation.
type ValidationResult struct {
	Fix                   *Fix
	InterfaceCompatible   bool
	TestsPass             bool
	ExecutionTimeImproved bool
	OriginalExecutionTime time.Duration
	NewExecutionTime      time.Duration
	Errors                []error
	TestOutput            string
	ValidatedAt           time.Time
}

// MockImplementation represents a mock implementation following the established pattern.
type MockImplementation struct {
	ComponentName            string
	InterfaceName            string
	Package                  string
	FilePath                 string
	Code                     string
	InterfaceMethods         []MethodSignature
	Status                   MockStatus
	RequiresManualCompletion bool
	GeneratedAt              time.Time
}

// MockStatus is an enumeration of mock implementation statuses.
type MockStatus int

const (
	MockStatusTemplate MockStatus = iota
	MockStatusComplete
	MockStatusValidated
)

// String returns the string representation of MockStatus.
func (s MockStatus) String() string {
	switch s {
	case MockStatusTemplate:
		return "Template"
	case MockStatusComplete:
		return "Complete"
	case MockStatusValidated:
		return "Validated"
	default:
		return "Unknown"
	}
}

// MethodSignature represents a method signature from an interface.
type MethodSignature struct {
	Name       string
	Parameters []Parameter
	Returns    []Return
	Receiver   string
}

// Parameter represents a function parameter.
type Parameter struct {
	Name string
	Type string
}

// Return represents a return value.
type Return struct {
	Name string
	Type string
}

// AnalysisReport represents the aggregated results of the analysis.
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

// AnalysisSummary represents summary statistics for an analysis.
type AnalysisSummary struct {
	TotalFiles       int
	TotalFunctions   int
	TotalIssues      int
	IssuesByType     map[IssueType]int
	IssuesBySeverity map[Severity]int
}

// ReportFormat is an enumeration of report formats.
type ReportFormat int

const (
	FormatJSON ReportFormat = iota
	FormatHTML
	FormatMarkdown
	FormatPlain
)

// String returns the string representation of ReportFormat.
func (f ReportFormat) String() string {
	switch f {
	case FormatJSON:
		return "json"
	case FormatHTML:
		return "html"
	case FormatMarkdown:
		return "markdown"
	case FormatPlain:
		return "plain"
	default:
		return "unknown"
	}
}
