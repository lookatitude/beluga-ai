# Reporter Interface Contract

## Interface: Reporter

### Purpose
Interface for generating analysis reports in various formats.

### Methods

#### GenerateReport
```go
GenerateReport(ctx context.Context, report *AnalysisReport, format ReportFormat) ([]byte, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation
- `report` (*AnalysisReport): Analysis report data
- `format` (ReportFormat): Output format (JSON, HTML, Markdown, Plain)

**Output**:
- `[]byte`: Generated report content
- `error`: Error if generation fails

**Behavior**:
- Generates report in specified format
- Includes summary statistics, categorized issues, and recommendations
- Returns report as bytes

**Errors**:
- `ErrInvalidFormat`: Format is not supported
- `ErrGenerationFailed`: Report generation failed

#### GenerateSummary
```go
GenerateSummary(ctx context.Context, report *AnalysisReport) (string, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation
- `report` (*AnalysisReport): Analysis report data

**Output**:
- `string`: Summary text
- `error`: Error if generation fails

**Behavior**:
- Generates human-readable summary
- Includes key statistics and top issues
- Returns summary as string

**Errors**:
- `ErrGenerationFailed`: Summary generation failed

#### GeneratePackageReport
```go
GeneratePackageReport(ctx context.Context, packageAnalysis *PackageAnalysis, format ReportFormat) ([]byte, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation
- `packageAnalysis` (*PackageAnalysis): Package analysis data
- `format` (ReportFormat): Output format

**Output**:
- `[]byte`: Generated package report
- `error`: Error if generation fails

**Behavior**:
- Generates report for single package
- Includes file-level and function-level issues
- Returns report as bytes

**Errors**:
- `ErrInvalidFormat`: Format is not supported
- `ErrGenerationFailed`: Report generation failed

## Report Formats

### ReportFormat
Enumeration of supported report formats.

**Values**:
- `FormatJSON`: JSON format for programmatic consumption
- `FormatHTML`: HTML format for web viewing
- `FormatMarkdown`: Markdown format for documentation
- `FormatPlain`: Plain text format for terminal output

### Report Structure

#### JSON Format
```json
{
  "summary": {
    "packages_analyzed": 14,
    "files_analyzed": 120,
    "functions_analyzed": 500,
    "issues_found": 45,
    "fixes_applied": 30,
    "fixes_failed": 2,
    "execution_time": "25s"
  },
  "issues_by_type": {
    "InfiniteLoop": 2,
    "MissingTimeout": 15,
    "LargeIteration": 8,
    "ActualImplementationUsage": 20
  },
  "issues_by_severity": {
    "Critical": 2,
    "High": 18,
    "Medium": 20,
    "Low": 5
  },
  "packages": [
    {
      "name": "pkg/llms",
      "issues": 10,
      "files": ["llms_test.go", "advanced_test.go"]
    }
  ],
  "generated_at": "2025-01-27T10:00:00Z"
}
```

#### HTML Format
- Interactive report with expandable sections
- Color-coded severity indicators
- Clickable links to source code
- Charts and graphs for statistics

#### Markdown Format
- Structured markdown document
- Code blocks for issue locations
- Tables for statistics
- Links to source files

#### Plain Format
- Terminal-friendly output
- Color-coded severity (if terminal supports)
- Tabular statistics
- Line-by-line issue listings

## Data Types

### AnalysisReport
```go
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
```

