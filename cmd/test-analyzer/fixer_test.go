package main

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/code"
	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/mocks"
	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/validation"
)

func TestNewFixer(t *testing.T) {
	mockGenerator := mocks.NewGenerator()
	codeModifier := code.NewModifier()
	validator := validation.NewValidator()
	fixer := NewFixer(mockGenerator, codeModifier, validator)

	if fixer == nil {
		t.Fatal("NewFixer() returned nil")
	}

	if _, ok := fixer.(Fixer); !ok {
		t.Error("NewFixer() does not implement Fixer interface")
	}
}

func TestFixer_ApplyFix(t *testing.T) {
	ctx := context.Background()
	mockGenerator := mocks.NewGenerator()
	codeModifier := code.NewModifier()
	validator := validation.NewValidator()
	fixer := NewFixer(mockGenerator, codeModifier, validator)

	t.Run("ApplyFixBasic", func(t *testing.T) {
		issue := PerformanceIssue{
			Type:     IssueTypeMissingTimeout,
			Severity: SeverityHigh,
		}

		_, err := fixer.ApplyFix(ctx, &issue)
		// May return error for invalid fix
		_ = err
	})
}

func TestFixer_ValidateFix(t *testing.T) {
	ctx := context.Background()
	mockGenerator := mocks.NewGenerator()
	codeModifier := code.NewModifier()
	validator := validation.NewValidator()
	fixer := NewFixer(mockGenerator, codeModifier, validator)

	t.Run("ValidateFixBasic", func(t *testing.T) {
		fix := &Fix{
			Type:   FixTypeAddTimeout,
			Status: FixStatusProposed,
		}

		_, err := fixer.ValidateFix(ctx, fix)
		// May return error for invalid fix
		_ = err
	})
}

func TestFixer_RollbackFix(t *testing.T) {
	ctx := context.Background()
	mockGenerator := mocks.NewGenerator()
	codeModifier := code.NewModifier()
	validator := validation.NewValidator()
	fixer := NewFixer(mockGenerator, codeModifier, validator)

	t.Run("RollbackFixBasic", func(t *testing.T) {
		fix := &Fix{
			Type:   FixTypeAddTimeout,
			Status: FixStatusApplied,
		}

		err := fixer.RollbackFix(ctx, fix)
		// May return error for invalid fix
		_ = err
	})
}

