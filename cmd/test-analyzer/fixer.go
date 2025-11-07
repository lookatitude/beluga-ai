package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/code"
	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/fixes"
	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/mocks"
	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/validation"
)

// Fixer is the interface for applying automated fixes to identified performance issues.
type Fixer interface {
	// ApplyFix applies a fix to an issue and returns the fix result.
	ApplyFix(ctx context.Context, issue *PerformanceIssue) (*Fix, error)

	// ValidateFix validates a fix using dual validation (interface compatibility + test execution).
	ValidateFix(ctx context.Context, fix *Fix) (*ValidationResult, error)

	// RollbackFix rolls back an applied fix by restoring the file from backup.
	RollbackFix(ctx context.Context, fix *Fix) error
}

// fixer implements the Fixer interface.
type fixer struct {
	mockGenerator  mocks.MockGenerator
	codeModifier   code.CodeModifier
	validator      validation.FixValidator
}

// NewFixer creates a new Fixer instance with the given dependencies.
func NewFixer(mockGenerator mocks.MockGenerator, codeModifier code.CodeModifier, validator validation.FixValidator) Fixer {
	return &fixer{
		mockGenerator: mockGenerator,
		codeModifier: codeModifier,
		validator:    validator,
	}
}

// ApplyFix implements Fixer.ApplyFix.
func (f *fixer) ApplyFix(ctx context.Context, issue *PerformanceIssue) (*Fix, error) {
	// Determine fix type based on issue type
	fixType := f.determineFixType(issue)
	if fixType == FixTypeUnknown {
		return nil, fmt.Errorf("cannot determine fix type for issue: %v", issue.Type)
	}

	// Create backup before modification
	backupPath, err := f.codeModifier.CreateBackup(ctx, issue.Location.File)
	if err != nil {
		return nil, fmt.Errorf("creating backup: %w", err)
	}

	// Generate code changes based on fix type
	changes, err := f.generateCodeChanges(ctx, issue, fixType)
	if err != nil {
		return nil, fmt.Errorf("generating code changes: %w", err)
	}

	// Apply code changes
	for _, change := range changes {
		// Convert main CodeChange to internal CodeChange
		internalChange := &code.CodeChange{
			File:        change.File,
			LineStart:   change.LineStart,
			LineEnd:     change.LineEnd,
			OldCode:     change.OldCode,
			NewCode:     change.NewCode,
			Description: change.Description,
		}
		if err := f.codeModifier.ApplyCodeChange(ctx, internalChange); err != nil {
			// Rollback on error
			_ = f.rollbackFromBackup(ctx, backupPath, issue.Location.File)
			return nil, fmt.Errorf("applying code change: %w", err)
		}
	}

	// Create fix object
	fix := &Fix{
		Issue:      issue,
		Type:       fixType,
		Changes:    changes,
		Status:     FixStatusApplied,
		BackupPath: backupPath,
		AppliedAt:  time.Now(),
	}

	return fix, nil
}

// ValidateFix implements Fixer.ValidateFix.
func (f *fixer) ValidateFix(ctx context.Context, fix *Fix) (*ValidationResult, error) {
	// Convert main Fix to internal Fix
	internalFix := &validation.Fix{
		Type:       fix.Type.String(),
		Status:     fix.Status.String(),
		BackupPath: fix.BackupPath,
		AppliedAt:  fix.AppliedAt,
	}
	// Convert Changes
	for _, change := range fix.Changes {
		internalFix.Changes = append(internalFix.Changes, validation.CodeChange{
			File:        change.File,
			LineStart:   change.LineStart,
			LineEnd:     change.LineEnd,
			OldCode:     change.OldCode,
			NewCode:     change.NewCode,
			Description: change.Description,
		})
	}

	// Validate
	internalResult, err := f.validator.ValidateFix(ctx, internalFix)
	if err != nil {
		return nil, err
	}

	// Convert internal ValidationResult to main ValidationResult
	result := &ValidationResult{
		Fix:                   fix,
		InterfaceCompatible:    internalResult.InterfaceCompatible,
		TestsPass:              internalResult.TestsPass,
		ExecutionTimeImproved:  internalResult.ExecutionTimeImproved,
		OriginalExecutionTime:  internalResult.OriginalExecutionTime,
		NewExecutionTime:       internalResult.NewExecutionTime,
		Errors:                 internalResult.Errors,
		TestOutput:             internalResult.TestOutput,
		ValidatedAt:            internalResult.ValidatedAt,
	}

	return result, nil
}

// RollbackFix implements Fixer.RollbackFix.
func (f *fixer) RollbackFix(ctx context.Context, fix *Fix) error {
	if fix.BackupPath == "" {
		return fmt.Errorf("no backup path available")
	}

	// Restore from backup
	if err := f.rollbackFromBackup(ctx, fix.BackupPath, fix.Issue.Location.File); err != nil {
		return fmt.Errorf("rolling back fix: %w", err)
	}

	fix.Status = FixStatusRolledBack
	return nil
}

// determineFixType determines the appropriate fix type for an issue.
func (f *fixer) determineFixType(issue *PerformanceIssue) FixType {
	switch issue.Type {
	case IssueTypeInfiniteLoop:
		return FixTypeAddLoopExit
	case IssueTypeMissingTimeout:
		return FixTypeAddTimeout
	case IssueTypeLargeIteration:
		return FixTypeReduceIterations
	case IssueTypeSleepDelay:
		return FixTypeOptimizeSleep
	case IssueTypeActualImplementationUsage:
		return FixTypeReplaceWithMock
	case IssueTypeMissingMock:
		return FixTypeCreateMock
	case IssueTypeMixedMockRealUsage:
		return FixTypeReplaceWithMock
	default:
		return FixTypeUnknown
	}
}

// generateCodeChanges generates code changes for a fix.
func (f *fixer) generateCodeChanges(ctx context.Context, issue *PerformanceIssue, fixType FixType) ([]CodeChange, error) {
	var changes []CodeChange

	switch fixType {
	case FixTypeAddTimeout:
		// Extract timeout duration from issue context or use default
		timeoutDuration := "5s"
		if duration, ok := issue.Context["timeout_duration"].(string); ok {
			timeoutDuration = duration
		}
		
		oldCode, newCode, err := fixes.GenerateTimeoutFix(ctx, issue.Location.Function, issue.Location.LineStart, issue.Location.LineEnd, timeoutDuration)
		if err != nil {
			return nil, fmt.Errorf("generating timeout fix: %w", err)
		}
		changes = append(changes, CodeChange{
			File:        issue.Location.File,
			LineStart:   issue.Location.LineStart,
			LineEnd:     issue.Location.LineEnd,
			OldCode:     oldCode,
			NewCode:     newCode,
			Description: fmt.Sprintf("Add timeout to %s", issue.Location.Function),
		})

	case FixTypeAddLoopExit:
		oldCode, newCode, err := fixes.GenerateLoopExitFix(ctx, issue.Location.LineStart, issue.Location.LineEnd)
		if err != nil {
			return nil, fmt.Errorf("generating loop exit fix: %w", err)
		}
		changes = append(changes, CodeChange{
			File:        issue.Location.File,
			LineStart:   issue.Location.LineStart,
			LineEnd:     issue.Location.LineEnd,
			OldCode:     oldCode,
			NewCode:     newCode,
			Description: fmt.Sprintf("Add exit condition to infinite loop in %s", issue.Location.Function),
		})

	case FixTypeReduceIterations:
		// Extract iteration counts from issue context
		oldCount := 1000
		newCount := 100
		if count, ok := issue.Context["iteration_count"].(int); ok {
			oldCount = count
			newCount = count / 10 // Reduce by 90%
			if newCount < 10 {
				newCount = 10
			}
		}
		
		oldCode, newCode, err := fixes.GenerateIterationFix(ctx, oldCount, newCount, issue.Location.LineStart, issue.Location.LineEnd)
		if err != nil {
			return nil, fmt.Errorf("generating iteration fix: %w", err)
		}
		changes = append(changes, CodeChange{
			File:        issue.Location.File,
			LineStart:   issue.Location.LineStart,
			LineEnd:     issue.Location.LineEnd,
			OldCode:     oldCode,
			NewCode:     newCode,
			Description: fmt.Sprintf("Reduce iterations from %d to %d in %s", oldCount, newCount, issue.Location.Function),
		})

	case FixTypeOptimizeSleep:
		// Extract sleep duration from issue context
		oldDuration := 200 * time.Millisecond
		newDuration := 10 * time.Millisecond
		if dur, ok := issue.Context["sleep_duration"].(time.Duration); ok {
			oldDuration = dur
			newDuration = dur / 20 // Reduce by 95%
			if newDuration < time.Millisecond {
				newDuration = time.Millisecond
			}
		}
		
		oldCode, newCode, err := fixes.GenerateSleepFix(ctx, oldDuration, newDuration, issue.Location.LineStart, issue.Location.LineEnd)
		if err != nil {
			return nil, fmt.Errorf("generating sleep fix: %w", err)
		}
		changes = append(changes, CodeChange{
			File:        issue.Location.File,
			LineStart:   issue.Location.LineStart,
			LineEnd:     issue.Location.LineEnd,
			OldCode:     oldCode,
			NewCode:     newCode,
			Description: fmt.Sprintf("Optimize sleep duration from %v to %v in %s", oldDuration, newDuration, issue.Location.Function),
		})

	case FixTypeReplaceWithMock:
		// Extract component name from issue context
		componentName := "component"
		if name, ok := issue.Context["component_name"].(string); ok {
			componentName = name
		}
		mockName := strings.ToLower(componentName) + "Mock"
		
		oldCode, newCode, err := fixes.GenerateMockReplacementFix(ctx, componentName, mockName, issue.Location.LineStart, issue.Location.LineEnd)
		if err != nil {
			return nil, fmt.Errorf("generating mock replacement fix: %w", err)
		}
		changes = append(changes, CodeChange{
			File:        issue.Location.File,
			LineStart:   issue.Location.LineStart,
			LineEnd:     issue.Location.LineEnd,
			OldCode:     oldCode,
			NewCode:     newCode,
			Description: fmt.Sprintf("Replace %s with mock in %s", componentName, issue.Location.Function),
		})

	case FixTypeCreateMock:
		// Extract interface information from issue context
		componentName := "component"
		interfaceName := "Interface"
		packagePath := ""
		if name, ok := issue.Context["component_name"].(string); ok {
			componentName = name
		}
		if iface, ok := issue.Context["interface_name"].(string); ok {
			interfaceName = iface
		}
		if pkg, ok := issue.Context["package_path"].(string); ok {
			packagePath = pkg
		}
		
		mockCode, err := fixes.GenerateMockCreationFix(ctx, componentName, interfaceName, packagePath)
		if err != nil {
			return nil, fmt.Errorf("generating mock creation fix: %w", err)
		}
		// For mock creation, we need to create a new file or append to existing mock file
		// This is a simplified version - in practice, this would be more complex
		mockFile := issue.Location.File[:strings.LastIndex(issue.Location.File, "_test.go")] + "_mock.go"
		changes = append(changes, CodeChange{
			File:        mockFile,
			LineStart:   1,
			LineEnd:     1,
			OldCode:     "",
			NewCode:     mockCode,
			Description: fmt.Sprintf("Create mock for %s", componentName),
		})

	default:
		return nil, fmt.Errorf("unsupported fix type: %v", fixType)
	}

	return changes, nil
}

// rollbackFromBackup restores a file from backup.
func (f *fixer) rollbackFromBackup(ctx context.Context, backupPath, filePath string) error {
	// Read backup file
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("reading backup: %w", err)
	}

	// Write backup data to original file
	if err := os.WriteFile(filePath, backupData, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

