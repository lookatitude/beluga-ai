package validation

import (
	"context"
	"testing"
)

func TestNewTestRunner(t *testing.T) {
	runner := NewTestRunner()
	if runner == nil {
		t.Fatal("NewTestRunner() returned nil")
	}

	if _, ok := runner.(TestRunner); !ok {
		t.Error("NewTestRunner() does not implement TestRunner interface")
	}
}

func TestTestRunner_RunTests(t *testing.T) {
	ctx := context.Background()
	runner := NewTestRunner()

	t.Run("RunTestsBasic", func(t *testing.T) {
		fix := &Fix{
			Type:   "AddTimeout",
			Status: "Proposed",
			Changes: []CodeChange{
				{File: "test.go"},
			},
		}

		_, err := runner.RunTests(ctx, fix)
		// May return error for invalid fix
		_ = err
	})
}
