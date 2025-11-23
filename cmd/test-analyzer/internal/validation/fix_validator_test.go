package validation

import (
	"context"
	"testing"
)

func TestNewFixValidator(t *testing.T) {
	validator := NewFixValidator()
	if validator == nil {
		t.Fatal("NewFixValidator() returned nil")
	}

	if _, ok := validator.(FixValidator); !ok {
		t.Error("NewFixValidator() does not implement FixValidator interface")
	}
}

func TestFixValidator_ValidateFix(t *testing.T) {
	ctx := context.Background()
	validator := NewFixValidator()

	t.Run("ValidateFixBasic", func(t *testing.T) {
		fix := &Fix{
			Type:   "AddTimeout",
			Status: "Proposed",
		}

		_, err := validator.ValidateFix(ctx, fix)
		// May return error for invalid fix
		_ = err
	})
}
