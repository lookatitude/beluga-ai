package dapr

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/workflow"
)

// mockStateClient is an in-memory mock of the StateClient interface.
type mockStateClient struct {
	mu    sync.RWMutex
	store map[string][]byte
	err   error
}

func newMockStateClient() *mockStateClient {
	return &mockStateClient{store: make(map[string][]byte)}
}

func (c *mockStateClient) SaveState(_ context.Context, _, key string, data []byte, _ map[string]string, _ ...any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	c.store[key] = data
	return nil
}

func (c *mockStateClient) GetState(_ context.Context, _, key string, _ map[string]string) (*StateItem, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.err != nil {
		return nil, c.err
	}
	data, ok := c.store[key]
	if !ok {
		return &StateItem{Key: key}, nil
	}
	return &StateItem{Key: key, Value: data}, nil
}

func (c *mockStateClient) DeleteState(_ context.Context, _, key string, _ map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	delete(c.store, key)
	return nil
}

func TestNew(t *testing.T) {
	t.Run("nil client returns error", func(t *testing.T) {
		_, err := New(Config{})
		if err == nil {
			t.Fatal("expected error for nil client")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		store, err := New(Config{Client: newMockStateClient()})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if store == nil {
			t.Fatal("expected non-nil store")
		}
	})

	t.Run("default store name", func(t *testing.T) {
		store, _ := New(Config{Client: newMockStateClient()})
		if store.storeName != "statestore" {
			t.Errorf("expected 'statestore', got %q", store.storeName)
		}
	})

	t.Run("custom store name", func(t *testing.T) {
		store, _ := New(Config{Client: newMockStateClient(), StoreName: "custom"})
		if store.storeName != "custom" {
			t.Errorf("expected 'custom', got %q", store.storeName)
		}
	})
}

func TestSaveAndLoad(t *testing.T) {
	client := newMockStateClient()
	store := NewWithClient(client, "")
	ctx := context.Background()

	state := workflow.WorkflowState{
		WorkflowID: "wf-1",
		RunID:      "run-1",
		Status:     workflow.StatusRunning,
		Input:      "test input",
	}

	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, "wf-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil state")
	}
	if loaded.WorkflowID != "wf-1" {
		t.Errorf("expected 'wf-1', got %q", loaded.WorkflowID)
	}
	if loaded.Status != workflow.StatusRunning {
		t.Errorf("expected running, got %s", loaded.Status)
	}
}

func TestLoadNotFound(t *testing.T) {
	client := newMockStateClient()
	store := NewWithClient(client, "")

	loaded, err := store.Load(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil for nonexistent workflow")
	}
}

func TestSaveEmptyID(t *testing.T) {
	client := newMockStateClient()
	store := NewWithClient(client, "")

	err := store.Save(context.Background(), workflow.WorkflowState{})
	if err == nil {
		t.Fatal("expected error for empty workflow ID")
	}
}

func TestSaveOverwrite(t *testing.T) {
	client := newMockStateClient()
	store := NewWithClient(client, "")
	ctx := context.Background()

	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1", Status: workflow.StatusRunning})
	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1", Status: workflow.StatusCompleted})

	loaded, _ := store.Load(ctx, "wf-1")
	if loaded.Status != workflow.StatusCompleted {
		t.Errorf("expected completed, got %s", loaded.Status)
	}
}

func TestList(t *testing.T) {
	client := newMockStateClient()
	store := NewWithClient(client, "")
	ctx := context.Background()

	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1", Status: workflow.StatusRunning})
	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-2", Status: workflow.StatusCompleted})
	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-3", Status: workflow.StatusRunning})

	// List all.
	all, err := store.List(ctx, workflow.WorkflowFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 workflows, got %d", len(all))
	}

	// Filter by status.
	running, err := store.List(ctx, workflow.WorkflowFilter{Status: workflow.StatusRunning})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(running) != 2 {
		t.Errorf("expected 2 running workflows, got %d", len(running))
	}

	// With limit.
	limited, err := store.List(ctx, workflow.WorkflowFilter{Limit: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(limited) != 1 {
		t.Errorf("expected 1 workflow, got %d", len(limited))
	}
}

func TestListEmpty(t *testing.T) {
	client := newMockStateClient()
	store := NewWithClient(client, "")

	results, err := store.List(context.Background(), workflow.WorkflowFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestDelete(t *testing.T) {
	client := newMockStateClient()
	store := NewWithClient(client, "")
	ctx := context.Background()

	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1"})
	if err := store.Delete(ctx, "wf-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	loaded, _ := store.Load(ctx, "wf-1")
	if loaded != nil {
		t.Error("expected nil after delete")
	}
}

func TestSaveWithHistory(t *testing.T) {
	client := newMockStateClient()
	store := NewWithClient(client, "")
	ctx := context.Background()

	state := workflow.WorkflowState{
		WorkflowID: "wf-1",
		Status:     workflow.StatusRunning,
		History: []workflow.HistoryEvent{
			{ID: 1, Type: workflow.EventWorkflowStarted},
			{ID: 2, Type: workflow.EventActivityStarted, ActivityName: "task1"},
		},
	}

	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, _ := store.Load(ctx, "wf-1")
	if len(loaded.History) != 2 {
		t.Errorf("expected 2 history events, got %d", len(loaded.History))
	}
}

func TestSaveError(t *testing.T) {
	client := newMockStateClient()
	client.err = fmt.Errorf("save failed")
	store := NewWithClient(client, "")

	err := store.Save(context.Background(), workflow.WorkflowState{WorkflowID: "wf-1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInterfaceCompliance(t *testing.T) {
	var _ workflow.WorkflowStore = (*Store)(nil)
}
