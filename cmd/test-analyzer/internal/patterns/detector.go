package patterns

import (
	"context"
)

// PerformanceIssue represents a detected issue.
type PerformanceIssue struct {
	Type        string
	Severity    string
	Location    Location
	Description string
	Context     map[string]interface{}
	Fixable     bool
}

// Location represents the location of an issue.
type Location struct {
	Package     string
	File        string
	Function    string
	LineStart   int
	LineEnd     int
	ColumnStart int
	ColumnEnd   int
}

// TestFunction represents a test function.
type TestFunction struct {
	Name                   string
	Type                   string
	File                   *TestFileInfo
	LineStart              int
	LineEnd                int
	HasTimeout             bool
	TimeoutDuration        int64
	ExecutionTime          int64
	UsesActualImplementation bool
	UsesMocks              bool
	MixedUsage             bool
	AST                    interface{} // *ast.FuncDecl - using interface{} to avoid importing go/ast in patterns package
}

// TestFileInfo contains file information for test functions.
type TestFileInfo struct {
	Path string
}

// PatternDetector is the interface for detecting specific performance patterns in test code.
type PatternDetector interface {
	// DetectAll detects all performance issues in a test function.
	DetectAll(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
}

// detector implements the PatternDetector interface.
type detector struct {
	infiniteLoopDetector      InfiniteLoopDetector
	timeoutDetector           TimeoutDetector
	iterationsDetector        IterationsDetector
	complexityDetector        ComplexityDetector
	implementationDetector    ImplementationDetector
	mocksDetector             MocksDetector
	sleepDetector             SleepDetector
	benchmarkDetector         BenchmarkDetector
	testTypeDetector          TestTypeDetector
}

// NewDetector creates a new PatternDetector instance.
func NewDetector() PatternDetector {
	return &detector{
		infiniteLoopDetector:   NewInfiniteLoopDetector(),
		timeoutDetector:        NewTimeoutDetector(),
		iterationsDetector:     NewIterationsDetector(),
		complexityDetector:     NewComplexityDetector(),
		implementationDetector: NewImplementationDetector(),
		mocksDetector:          NewMocksDetector(),
		sleepDetector:          NewSleepDetector(),
		benchmarkDetector:      NewBenchmarkDetector(),
		testTypeDetector:       NewTestTypeDetector(),
	}
}

// DetectAll implements PatternDetector.DetectAll.
func (d *detector) DetectAll(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error) {
	var allIssues []PerformanceIssue

	// Run all detectors
	if issues, err := d.infiniteLoopDetector.Detect(ctx, function); err == nil {
		allIssues = append(allIssues, issues...)
	}
	if issues, err := d.timeoutDetector.Detect(ctx, function); err == nil {
		allIssues = append(allIssues, issues...)
	}
	if issues, err := d.iterationsDetector.Detect(ctx, function); err == nil {
		allIssues = append(allIssues, issues...)
	}
	if issues, err := d.complexityDetector.Detect(ctx, function); err == nil {
		allIssues = append(allIssues, issues...)
	}
	if issues, err := d.implementationDetector.Detect(ctx, function); err == nil {
		allIssues = append(allIssues, issues...)
	}
	if issues, err := d.mocksDetector.Detect(ctx, function); err == nil {
		allIssues = append(allIssues, issues...)
	}
	if issues, err := d.sleepDetector.Detect(ctx, function); err == nil {
		allIssues = append(allIssues, issues...)
	}
	if issues, err := d.benchmarkDetector.Detect(ctx, function); err == nil {
		allIssues = append(allIssues, issues...)
	}

	return allIssues, nil
}
