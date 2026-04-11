package sleeptime

import (
	"context"
	"testing"
	"time"
)

// mockTask implements the Task interface for testing.
type mockTask struct {
	name      string
	priority  Priority
	shouldRun bool
	runResult TaskResult
	runErr    error
	runCalled int
}

var _ Task = (*mockTask)(nil)

func (m *mockTask) Name() string                                     { return m.name }
func (m *mockTask) Priority() Priority                               { return m.priority }
func (m *mockTask) ShouldRun(_ context.Context, _ SessionState) bool { return m.shouldRun }
func (m *mockTask) Run(_ context.Context, _ SessionState) (TaskResult, error) {
	m.runCalled++
	return m.runResult, m.runErr
}

func TestTaskRegistry(t *testing.T) {
	// Save and restore registry state.
	origRegistry := make(map[string]TaskFactory, len(taskRegistry))
	for k, v := range taskRegistry {
		origRegistry[k] = v
	}
	defer func() {
		taskMu.Lock()
		taskRegistry = origRegistry
		taskMu.Unlock()
	}()

	t.Run("register and create", func(t *testing.T) {
		RegisterTask("test_task", func(cfg map[string]any) (Task, error) {
			return &mockTask{name: "test_task", shouldRun: true}, nil
		})

		task, err := NewTask("test_task", nil)
		if err != nil {
			t.Fatalf("NewTask() error = %v", err)
		}
		if task.Name() != "test_task" {
			t.Errorf("Name() = %q, want %q", task.Name(), "test_task")
		}
	})

	t.Run("unknown task", func(t *testing.T) {
		_, err := NewTask("nonexistent", nil)
		if err == nil {
			t.Fatal("expected error for unknown task")
		}
	})

	t.Run("list tasks", func(t *testing.T) {
		RegisterTask("alpha_task", func(cfg map[string]any) (Task, error) {
			return &mockTask{name: "alpha_task"}, nil
		})
		RegisterTask("beta_task", func(cfg map[string]any) (Task, error) {
			return &mockTask{name: "beta_task"}, nil
		})

		names := ListTasks()
		if len(names) < 2 {
			t.Fatalf("ListTasks() returned %d names, want >= 2", len(names))
		}

		// Verify sorted order.
		for i := 1; i < len(names); i++ {
			if names[i] < names[i-1] {
				t.Errorf("ListTasks() not sorted: %v", names)
				break
			}
		}
	})
}

func TestSessionState(t *testing.T) {
	state := SessionState{
		SessionID:    "sess-1",
		AgentID:      "agent-1",
		TurnCount:    15,
		LastActivity: time.Now(),
		Metadata:     map[string]any{"key": "value"},
	}

	if state.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", state.SessionID, "sess-1")
	}
	if state.TurnCount != 15 {
		t.Errorf("TurnCount = %d, want %d", state.TurnCount, 15)
	}
}

func TestTaskResult(t *testing.T) {
	result := TaskResult{
		TaskName:       "test",
		Success:        true,
		Duration:       100 * time.Millisecond,
		ItemsProcessed: 5,
	}

	if !result.Success {
		t.Error("expected success")
	}
	if result.Error != "" {
		t.Errorf("Error = %q, want empty", result.Error)
	}
}
