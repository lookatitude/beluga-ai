package validation

import (
	"context"
	"fmt"
	"time"
)

// FixValidator is the interface for validating fixes.
type FixValidator interface {
	// ValidateFix validates a fix using dual validation.
	ValidateFix(ctx context.Context, fix *Fix) (*ValidationResult, error)
}

// fixValidator implements FixValidator.
type fixValidator struct {
	interfaceChecker InterfaceChecker
	testRunner       TestRunner
}

// NewFixValidator creates a new FixValidator instance.
func NewFixValidator() FixValidator {
	return &fixValidator{
		interfaceChecker: NewInterfaceChecker(),
		testRunner:       NewTestRunner(),
	}
}

// ValidateFix implements FixValidator.ValidateFix.
func (v *fixValidator) ValidateFix(ctx context.Context, fix *Fix) (*ValidationResult, error) {
	result := &ValidationResult{
		Fix:         fix,
		ValidatedAt: time.Now(),
	}

	// Step 1: Interface compatibility check (for mock-related fixes)
	if fix.Type == "ReplaceWithMock" || fix.Type == "CreateMock" {
		compatible, err := v.interfaceChecker.CheckInterfaceCompatibility(ctx, fix)
		if err != nil {
			result.Errors = append(result.Errors, err)
			return result, nil
		}
		result.InterfaceCompatible = compatible
		if !compatible {
			result.Errors = append(result.Errors, fmt.Errorf("interface compatibility check failed"))
			return result, nil
		}
	} else {
		// For non-mock fixes, skip interface check
		result.InterfaceCompatible = true
	}

	// Step 2: Test execution
	testResult, err := v.testRunner.RunTests(ctx, fix)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return result, nil
	}

	result.TestsPass = testResult.TestsPass
	result.TestOutput = testResult.Output
	result.OriginalExecutionTime = testResult.OriginalExecutionTime
	result.NewExecutionTime = testResult.NewExecutionTime
	result.ExecutionTimeImproved = testResult.NewExecutionTime < testResult.OriginalExecutionTime

	if !result.TestsPass {
		result.Errors = append(result.Errors, fmt.Errorf("tests failed after fix"))
	}

	return result, nil
}
