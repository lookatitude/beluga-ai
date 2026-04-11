package rl

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
)

func TestSimpleReward_Compute(t *testing.T) {
	reward := NewSimpleReward()
	ctx := context.Background()

	tests := []struct {
		name       string
		outcome    any
		numSteps   int
		wantReward float64
		wantErr    bool
	}{
		{
			name:       "bool success",
			outcome:    true,
			numSteps:   3,
			wantReward: 1.0,
		},
		{
			name:       "bool failure",
			outcome:    false,
			numSteps:   2,
			wantReward: -1.0,
		},
		{
			name:       "int success",
			outcome:    1,
			numSteps:   1,
			wantReward: 1.0,
		},
		{
			name:       "int failure",
			outcome:    0,
			numSteps:   1,
			wantReward: -1.0,
		},
		{
			name:       "int64 success",
			outcome:    int64(5),
			numSteps:   2,
			wantReward: 1.0,
		},
		{
			name:       "float64 success",
			outcome:    0.5,
			numSteps:   2,
			wantReward: 1.0,
		},
		{
			name:       "float64 failure",
			outcome:    -1.0,
			numSteps:   2,
			wantReward: -1.0,
		},
		{
			name:       "float32 success",
			outcome:    float32(0.1),
			numSteps:   1,
			wantReward: 1.0,
		},
		{
			name:       "nil outcome is failure",
			outcome:    nil,
			numSteps:   1,
			wantReward: -1.0,
		},
		{
			name:     "empty steps",
			outcome:  true,
			numSteps: 0,
		},
		{
			name:     "unsupported type",
			outcome:  "string",
			numSteps: 1,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := Episode{
				ID:      "test",
				Outcome: tt.outcome,
				Steps:   make([]Step, tt.numSteps),
			}

			rewards, err := reward.Compute(ctx, ep)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var coreErr *core.Error
				if !isErrorType(err, &coreErr) {
					t.Errorf("expected core.Error, got %T", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.numSteps == 0 {
				if rewards != nil {
					t.Errorf("expected nil rewards for empty steps, got %v", rewards)
				}
				return
			}

			if len(rewards) != tt.numSteps {
				t.Fatalf("rewards len = %d, want %d", len(rewards), tt.numSteps)
			}
			for i, r := range rewards {
				if r != tt.wantReward {
					t.Errorf("rewards[%d] = %v, want %v", i, r, tt.wantReward)
				}
			}
		})
	}
}

func TestSimpleReward_CustomValues(t *testing.T) {
	reward := &SimpleReward{
		SuccessReward: 10.0,
		FailureReward: -5.0,
	}

	ep := Episode{
		ID:      "custom",
		Outcome: true,
		Steps:   make([]Step, 2),
	}

	rewards, err := reward.Compute(context.Background(), ep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, r := range rewards {
		if r != 10.0 {
			t.Errorf("rewards[%d] = %v, want 10.0", i, r)
		}
	}
}

func TestNewSimpleReward_Defaults(t *testing.T) {
	r := NewSimpleReward()
	if r.SuccessReward != 1.0 {
		t.Errorf("SuccessReward = %v, want 1.0", r.SuccessReward)
	}
	if r.FailureReward != -1.0 {
		t.Errorf("FailureReward = %v, want -1.0", r.FailureReward)
	}
}

// isErrorType is a helper that uses errors.As without importing errors
// (to avoid shadowing in table tests).
func isErrorType(err error, target any) bool {
	// Use a simple type assertion chain.
	switch target.(type) {
	case **core.Error:
		_, ok := err.(*core.Error)
		return ok
	}
	return false
}
