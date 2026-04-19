package tasks

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/runtime/sleeptime"
)

func TestContradictionResolverTask_Name(t *testing.T) {
	task := NewContradictionResolverTask(nil)
	if got := task.Name(); got != "contradiction_resolver" {
		t.Errorf("Name() = %q, want %q", got, "contradiction_resolver")
	}
}

func TestContradictionResolverTask_Priority(t *testing.T) {
	task := NewContradictionResolverTask(nil)
	if got := task.Priority(); got != sleeptime.PriorityHigh {
		t.Errorf("Priority() = %d, want %d", got, sleeptime.PriorityHigh)
	}
}

func TestContradictionResolverTask_ShouldRun(t *testing.T) {
	tests := []struct {
		name      string
		minTurns  int
		turnCount int
		want      bool
	}{
		{"enough turns", 5, 10, true},
		{"exactly min turns", 5, 5, true},
		{"not enough turns", 5, 3, false},
		{"zero turns", 5, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := map[string]any{"min_turns": tt.minTurns}
			task := NewContradictionResolverTask(cfg)
			state := sleeptime.SessionState{TurnCount: tt.turnCount}

			got := task.ShouldRun(context.Background(), state)
			if got != tt.want {
				t.Errorf("ShouldRun() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContradictionResolverTask_Run(t *testing.T) {
	tests := []struct {
		name          string
		metadata      map[string]any
		wantProcessed int
	}{
		{
			name:          "nil metadata",
			metadata:      nil,
			wantProcessed: 0,
		},
		{
			name:          "no contradictions",
			metadata:      map[string]any{"name": "Alice"},
			wantProcessed: 0,
		},
		{
			name: "has fact keys",
			metadata: map[string]any{
				"fact_name":     "Alice",
				"fact_location": "Paris",
				"other":         "value",
			},
			wantProcessed: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := NewContradictionResolverTask(nil)
			state := sleeptime.SessionState{
				TurnCount: 10,
				Metadata:  tt.metadata,
			}

			result, err := task.Run(context.Background(), state)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if !result.Success {
				t.Error("expected success")
			}
			if result.ItemsProcessed != tt.wantProcessed {
				t.Errorf("ItemsProcessed = %d, want %d", result.ItemsProcessed, tt.wantProcessed)
			}
		})
	}
}

func TestContradictionResolverTask_Run_ContextCancelled(t *testing.T) {
	task := NewContradictionResolverTask(nil)
	state := sleeptime.SessionState{
		TurnCount: 10,
		Metadata: map[string]any{
			"fact_a": "1", "fact_b": "2", "fact_c": "3",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := task.Run(ctx, state)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestContradictionResolverTask_Config(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		task := NewContradictionResolverTask(nil)
		if task.minTurns != 5 {
			t.Errorf("minTurns = %d, want 5", task.minTurns)
		}
	})

	t.Run("custom values", func(t *testing.T) {
		task := NewContradictionResolverTask(map[string]any{
			"min_turns": 15,
			"max_age":   "8m",
		})
		if task.minTurns != 15 {
			t.Errorf("minTurns = %d, want 15", task.minTurns)
		}
	})
}

func TestContradictionResolverTask_Registry(t *testing.T) {
	task, err := sleeptime.NewTask("contradiction_resolver", nil)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	if task.Name() != "contradiction_resolver" {
		t.Errorf("Name() = %q, want %q", task.Name(), "contradiction_resolver")
	}
}
