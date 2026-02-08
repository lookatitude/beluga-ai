package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/workflow"
	"github.com/nats-io/nats.go/jetstream"
)

// mockKV implements kvStore for testing.
type mockKV struct {
	mu    sync.RWMutex
	store map[string][]byte
}

func newMockKV() *mockKV {
	return &mockKV{store: make(map[string][]byte)}
}

func (kv *mockKV) get(_ context.Context, key string) ([]byte, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	data, ok := kv.store[key]
	if !ok {
		return nil, jetstream.ErrKeyNotFound
	}
	return data, nil
}

func (kv *mockKV) put(_ context.Context, key string, value []byte) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.store[key] = value
	return nil
}

func (kv *mockKV) delete(_ context.Context, key string) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	delete(kv.store, key)
	return nil
}

func (kv *mockKV) keys(_ context.Context) ([]string, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	if len(kv.store) == 0 {
		return nil, fmt.Errorf("nats: no keys found")
	}
	keys := make([]string, 0, len(kv.store))
	for k := range kv.store {
		keys = append(keys, k)
	}
	return keys, nil
}

func TestSaveAndLoad(t *testing.T) {
	kv := newMockKV()
	store := newWithKV(kv)
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
	kv := newMockKV()
	store := newWithKV(kv)

	loaded, err := store.Load(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil for nonexistent workflow")
	}
}

func TestSaveEmptyID(t *testing.T) {
	kv := newMockKV()
	store := newWithKV(kv)

	err := store.Save(context.Background(), workflow.WorkflowState{})
	if err == nil {
		t.Fatal("expected error for empty workflow ID")
	}
}

func TestSaveOverwrite(t *testing.T) {
	kv := newMockKV()
	store := newWithKV(kv)
	ctx := context.Background()

	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1", Status: workflow.StatusRunning})
	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1", Status: workflow.StatusCompleted})

	loaded, _ := store.Load(ctx, "wf-1")
	if loaded.Status != workflow.StatusCompleted {
		t.Errorf("expected completed, got %s", loaded.Status)
	}
}

func TestList(t *testing.T) {
	kv := newMockKV()
	store := newWithKV(kv)
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
	kv := newMockKV()
	store := newWithKV(kv)

	results, err := store.List(context.Background(), workflow.WorkflowFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestDelete(t *testing.T) {
	kv := newMockKV()
	store := newWithKV(kv)
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

func TestDeleteNonexistent(t *testing.T) {
	kv := newMockKV()
	store := newWithKV(kv)

	// Delete of nonexistent key should not error (mapped from "not found").
	if err := store.Delete(context.Background(), "nonexistent"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestSaveWithHistory(t *testing.T) {
	kv := newMockKV()
	store := newWithKV(kv)
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

func TestClose(t *testing.T) {
	kv := newMockKV()
	store := newWithKV(kv)
	// Should not panic (owns=false, so no conn to close).
	store.Close()
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		err    error
		expect bool
	}{
		{nil, false},
		{fmt.Errorf("not found"), true},
		{fmt.Errorf("no keys found"), true},
		{jetstream.ErrKeyNotFound, true},
		{fmt.Errorf("other error"), false},
	}

	for _, tt := range tests {
		got := isNotFound(tt.err)
		if got != tt.expect {
			t.Errorf("isNotFound(%v) = %v, want %v", tt.err, got, tt.expect)
		}
	}
}

func TestJSONRoundTrip(t *testing.T) {
	state := workflow.WorkflowState{
		WorkflowID: "wf-rt",
		RunID:      "run-rt",
		Status:     workflow.StatusCompleted,
		Input:      "input data",
		Result:     "result data",
		Error:      "",
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded workflow.WorkflowState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded.WorkflowID != state.WorkflowID {
		t.Errorf("expected %q, got %q", state.WorkflowID, loaded.WorkflowID)
	}
	if loaded.Status != state.Status {
		t.Errorf("expected %q, got %q", state.Status, loaded.Status)
	}
}

func TestInterfaceCompliance(t *testing.T) {
	var _ workflow.WorkflowStore = (*Store)(nil)
}
