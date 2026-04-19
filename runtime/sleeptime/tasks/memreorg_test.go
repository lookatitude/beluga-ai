package tasks

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/runtime/sleeptime"
)

func TestMemoryReorgTask_Name(t *testing.T) {
	task := NewMemoryReorgTask(nil)
	if got := task.Name(); got != "memory_reorg" {
		t.Errorf("Name() = %q, want %q", got, "memory_reorg")
	}
}

func TestMemoryReorgTask_Priority(t *testing.T) {
	task := NewMemoryReorgTask(nil)
	if got := task.Priority(); got != sleeptime.PriorityNormal {
		t.Errorf("Priority() = %d, want %d", got, sleeptime.PriorityNormal)
	}
}

func TestMemoryReorgTask_ShouldRun(t *testing.T) {
	tests := []struct {
		name      string
		minTurns  int
		turnCount int
		want      bool
	}{
		{"enough turns", 10, 15, true},
		{"exactly min turns", 10, 10, true},
		{"not enough turns", 10, 5, false},
		{"zero turns", 10, 0, false},
		{"custom min turns", 3, 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := map[string]any{"min_turns": tt.minTurns}
			task := NewMemoryReorgTask(cfg)
			state := sleeptime.SessionState{TurnCount: tt.turnCount}

			got := task.ShouldRun(context.Background(), state)
			if got != tt.want {
				t.Errorf("ShouldRun() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemoryReorgTask_Run(t *testing.T) {
	task := NewMemoryReorgTask(map[string]any{"min_turns": 4})
	state := sleeptime.SessionState{TurnCount: 20}

	result, err := task.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.ItemsProcessed == 0 {
		t.Error("expected items to be processed")
	}
}

func TestMemoryReorgTask_Run_ContextCancelled(t *testing.T) {
	task := NewMemoryReorgTask(map[string]any{"min_turns": 2})
	state := sleeptime.SessionState{TurnCount: 1000}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := task.Run(ctx, state)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestMemoryReorgTask_Run_NothingToConsolidate(t *testing.T) {
	task := NewMemoryReorgTask(map[string]any{"min_turns": 10})
	state := sleeptime.SessionState{TurnCount: 10}

	result, err := task.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.ItemsProcessed != 5 {
		t.Errorf("ItemsProcessed = %d, want 5", result.ItemsProcessed)
	}
}

func TestMemoryReorgTask_Config(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		task := NewMemoryReorgTask(nil)
		if task.minTurns != 10 {
			t.Errorf("minTurns = %d, want 10", task.minTurns)
		}
	})

	t.Run("custom values", func(t *testing.T) {
		task := NewMemoryReorgTask(map[string]any{
			"min_turns": 20,
			"max_age":   "10m",
		})
		if task.minTurns != 20 {
			t.Errorf("minTurns = %d, want 20", task.minTurns)
		}
	})

	t.Run("invalid values ignored", func(t *testing.T) {
		task := NewMemoryReorgTask(map[string]any{
			"min_turns": -5,
			"max_age":   "invalid",
		})
		// Should keep defaults.
		if task.minTurns != 10 {
			t.Errorf("minTurns = %d, want 10 (default)", task.minTurns)
		}
	})
}

func TestMemoryReorgTask_Registry(t *testing.T) {
	task, err := sleeptime.NewTask("memory_reorg", nil)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	if task.Name() != "memory_reorg" {
		t.Errorf("Name() = %q, want %q", task.Name(), "memory_reorg")
	}
}
