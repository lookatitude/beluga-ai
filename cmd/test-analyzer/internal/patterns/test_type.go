package patterns

import (
	"context"
)

// TestTypeDetector determines test type (Unit/Integration/Load).
type TestTypeDetector interface {
	Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
}

// testTypeDetector implements TestTypeDetector.
type testTypeDetector struct{}

// NewTestTypeDetector creates a new TestTypeDetector.
func NewTestTypeDetector() TestTypeDetector {
	return &testTypeDetector{}
}

// Detect implements TestTypeDetector.Detect.
// Note: This detector doesn't return issues, but determines test type.
// It's included for completeness but the type is already determined during parsing.
func (d *testTypeDetector) Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error) {
	// Test type is already determined during AST parsing
	// This detector is a placeholder for future enhancements
	return nil, nil
}
