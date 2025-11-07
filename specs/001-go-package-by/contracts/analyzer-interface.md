# Analyzer Interface Contract

## Interface: Analyzer

### Purpose
Core interface for analyzing test files and detecting performance issues.

### Methods

#### AnalyzePackage
```go
AnalyzePackage(ctx context.Context, packagePath string) (*PackageAnalysis, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation and timeout
- `packagePath` (string): Path to package directory

**Output**:
- `*PackageAnalysis`: Analysis results for the package
- `error`: Error if analysis fails

**Behavior**:
- Analyzes all `*_test.go` files in the package
- Detects performance issues using AST parsing
- Returns categorized issues with locations and severity

**Errors**:
- `ErrPackageNotFound`: Package path does not exist
- `ErrInvalidPackage`: Package is not a valid Go package
- `ErrAnalysisTimeout`: Analysis exceeded timeout

#### AnalyzeFile
```go
AnalyzeFile(ctx context.Context, filePath string) (*FileAnalysis, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation and timeout
- `filePath` (string): Path to test file

**Output**:
- `*FileAnalysis`: Analysis results for the file
- `error`: Error if analysis fails

**Behavior**:
- Parses test file using AST
- Analyzes all test functions in the file
- Detects issues specific to the file

**Errors**:
- `ErrFileNotFound`: File path does not exist
- `ErrInvalidGoFile`: File is not a valid Go file
- `ErrParseError`: AST parsing failed

#### DetectIssues
```go
DetectIssues(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation and timeout
- `function` (*TestFunction): Test function to analyze

**Output**:
- `[]PerformanceIssue`: List of detected issues
- `error`: Error if detection fails

**Behavior**:
- Analyzes function AST for performance patterns
- Detects: infinite loops, missing timeouts, large iterations, sleep delays, actual implementation usage
- Returns categorized issues with severity

**Errors**:
- `ErrInvalidFunction`: Function is not a valid test function
- `ErrAnalysisError`: Analysis failed

## Interface: PatternDetector

### Purpose
Interface for detecting specific performance patterns in test code.

### Methods

#### DetectInfiniteLoops
```go
DetectInfiniteLoops(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation
- `function` (*TestFunction): Test function to analyze

**Output**:
- `[]PerformanceIssue`: List of infinite loop issues
- `error`: Error if detection fails

**Behavior**:
- Detects `for { }` patterns without proper exit conditions
- Flags as Critical severity

#### DetectMissingTimeouts
```go
DetectMissingTimeouts(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation
- `function` (*TestFunction): Test function to analyze

**Output**:
- `[]PerformanceIssue`: List of missing timeout issues
- `error`: Error if detection fails

**Behavior**:
- Checks for `context.WithTimeout`, `t.Deadline`, or test-level timeout
- Applies context-dependent thresholds (1s unit, 10s integration, 30s load)
- Flags as High severity for unit tests, Medium for integration tests

#### DetectLargeIterations
```go
DetectLargeIterations(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation
- `function` (*TestFunction): Test function to analyze

**Output**:
- `[]PerformanceIssue`: List of large iteration issues
- `error`: Error if detection fails

**Behavior**:
- Detects hardcoded iteration counts exceeding thresholds
- Applies context-dependent thresholds (100+ simple ops, 20+ complex ops)
- Detects operation complexity (network, I/O, DB calls)
- Flags as Medium severity for simple ops, High for complex ops

#### DetectActualImplementationUsage
```go
DetectActualImplementationUsage(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation
- `function` (*TestFunction): Test function to analyze

**Output**:
- `[]PerformanceIssue`: List of actual implementation usage issues
- `error`: Error if detection fails

**Behavior**:
- Detects direct instantiation of provider types
- Detects factory calls without mock registration
- Detects real API client initialization
- Detects actual file/database operations
- Applies context-aware handling (flag unit tests, allow integration tests)
- Flags as High severity for unit tests, Low for integration tests

#### DetectMissingMocks
```go
DetectMissingMocks(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation
- `function` (*TestFunction): Test function to analyze

**Output**:
- `[]PerformanceIssue`: List of missing mock issues
- `error`: Error if detection fails

**Behavior**:
- Analyzes interface usage in tests
- Compares against available mock implementations
- Checks `test_utils.go`, `internal/mock/`, `providers/mock/` directories
- Flags as Medium severity

## Data Types

### PackageAnalysis
```go
type PackageAnalysis struct {
    Package      string
    Files        []*FileAnalysis
    Issues       []PerformanceIssue
    Summary      *AnalysisSummary
    AnalyzedAt   time.Time
}
```

### FileAnalysis
```go
type FileAnalysis struct {
    File        *TestFile
    Functions   []*TestFunction
    Issues      []PerformanceIssue
    AnalyzedAt  time.Time
}
```

### AnalysisSummary
```go
type AnalysisSummary struct {
    TotalFiles      int
    TotalFunctions  int
    TotalIssues     int
    IssuesByType    map[IssueType]int
    IssuesBySeverity map[Severity]int
}
```

