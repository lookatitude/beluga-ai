package learning

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/schema"
)

// TestCase defines a single test case for validating a generated tool.
type TestCase struct {
	// Name is a descriptive name for the test case.
	Name string
	// Input is the input map to pass to the tool's Execute method.
	Input map[string]any
	// WantOutput is the expected output string. If empty, only checks that
	// execution succeeds without error.
	WantOutput string
	// WantError indicates whether the test case expects an error.
	WantError bool
}

// TestResult holds the outcome of running a single test case.
type TestResult struct {
	// Name is the test case name.
	Name string
	// Passed indicates whether the test case passed.
	Passed bool
	// GotOutput is the actual output from the tool.
	GotOutput string
	// Error is any error that occurred during execution.
	Error error
}

// ToolTester validates generated tools by running test inputs before registration.
// It ensures tools produce expected outputs and handle errors correctly.
type ToolTester struct {
	hooks Hooks
}

// TesterOption configures a ToolTester.
type TesterOption func(*ToolTester)

// WithTesterHooks sets lifecycle hooks on the tool tester.
func WithTesterHooks(h Hooks) TesterOption {
	return func(tt *ToolTester) {
		tt.hooks = h
	}
}

// NewToolTester creates a new ToolTester with the given options.
func NewToolTester(opts ...TesterOption) *ToolTester {
	tt := &ToolTester{}
	for _, opt := range opts {
		opt(tt)
	}
	return tt
}

// Test runs the given test cases against the tool and returns the results.
// All test cases are run regardless of individual failures.
func (tt *ToolTester) Test(ctx context.Context, t *DynamicTool, cases []TestCase) []TestResult {
	results := make([]TestResult, 0, len(cases))

	for _, tc := range cases {
		select {
		case <-ctx.Done():
			results = append(results, TestResult{
				Name:   tc.Name,
				Passed: false,
				Error:  ctx.Err(),
			})
			return results
		default:
		}

		result := tt.runCase(ctx, t, tc)
		results = append(results, result)
	}

	// Fire hook.
	if tt.hooks.OnToolTested != nil {
		allPassed := true
		for _, r := range results {
			if !r.Passed {
				allPassed = false
				break
			}
		}
		tt.hooks.OnToolTested(t.Name(), allPassed)
	}

	return results
}

// Validate runs test cases and returns an error if any test case fails.
// This is a convenience method for use in registration pipelines.
func (tt *ToolTester) Validate(ctx context.Context, t *DynamicTool, cases []TestCase) error {
	results := tt.Test(ctx, t, cases)

	var failures []string
	for _, r := range results {
		if !r.Passed {
			msg := fmt.Sprintf("test %q failed", r.Name)
			if r.Error != nil {
				msg += fmt.Sprintf(": %v", r.Error)
			}
			failures = append(failures, msg)
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("tool %q validation failed: %s", t.Name(), strings.Join(failures, "; "))
	}
	return nil
}

// runCase executes a single test case against the tool.
func (tt *ToolTester) runCase(ctx context.Context, t *DynamicTool, tc TestCase) TestResult {
	result, err := t.Execute(ctx, tc.Input)

	if tc.WantError {
		if err != nil {
			return TestResult{Name: tc.Name, Passed: true, Error: err}
		}
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Error:  fmt.Errorf("expected error but got none"),
		}
	}

	if err != nil {
		return TestResult{Name: tc.Name, Passed: false, Error: err}
	}

	// Extract text output from result.
	gotOutput := ""
	if result != nil && len(result.Content) > 0 {
		if tp, ok := result.Content[0].(schema.TextPart); ok {
			gotOutput = tp.Text
		}
	}

	if tc.WantOutput != "" && gotOutput != tc.WantOutput {
		return TestResult{
			Name:      tc.Name,
			Passed:    false,
			GotOutput: gotOutput,
			Error:     fmt.Errorf("output mismatch: want %q, got %q", tc.WantOutput, gotOutput),
		}
	}

	return TestResult{Name: tc.Name, Passed: true, GotOutput: gotOutput}
}
