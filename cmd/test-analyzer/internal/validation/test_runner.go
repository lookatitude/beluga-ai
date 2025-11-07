package validation

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TestRunner runs tests and measures execution time.
type TestRunner interface {
	RunTests(ctx context.Context, fix *Fix) (*TestResult, error)
}

// TestResult contains test execution results.
type TestResult struct {
	TestsPass              bool
	Output                 string
	OriginalExecutionTime  time.Duration
	NewExecutionTime       time.Duration
}

// testRunner implements TestRunner.
type testRunner struct{}

// NewTestRunner creates a new TestRunner.
func NewTestRunner() TestRunner {
	return &testRunner{}
}

// RunTests implements TestRunner.RunTests.
func (r *testRunner) RunTests(ctx context.Context, fix *Fix) (*TestResult, error) {
	// Determine package path from fix changes
	if len(fix.Changes) == 0 {
		return nil, fmt.Errorf("no changes in fix")
	}

	// Extract package path from first change file
	filePath := fix.Changes[0].File
	packagePath := filepath.Dir(filePath)

	// Run go test on the package
	cmd := exec.CommandContext(ctx, "go", "test", "-v", packagePath)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Measure execution time (simplified - would parse from go test output)
	executionTime := time.Duration(0)
	if strings.Contains(outputStr, "PASS") {
		executionTime = 50 * time.Millisecond // Placeholder
	}

	testsPass := err == nil && strings.Contains(outputStr, "PASS")

	return &TestResult{
		TestsPass:             testsPass,
		Output:                 outputStr,
		OriginalExecutionTime: 100 * time.Millisecond, // Placeholder
		NewExecutionTime:      executionTime,
	}, nil
}

