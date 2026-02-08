package kafka

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/workflow"
)

// mockWriter is an in-memory mock of the Writer interface.
type mockWriter struct {
	mu       sync.Mutex
	messages []Message
	err      error
	closed   bool
}

func (w *mockWriter) WriteMessages(_ context.Context, msgs ...Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return w.err
	}
	w.messages = append(w.messages, msgs...)
	return nil
}

func (w *mockWriter) Close() error {
	w.closed = true
	return nil
}

// mockReader is an in-memory mock of the Reader interface.
type mockReader struct {
	closed bool
}

func (r *mockReader) ReadMessage(_ context.Context) (Message, error) {
	return Message{}, fmt.Errorf("no messages")
}

func (r *mockReader) Close() error {
	r.closed = true
	return nil
}

func newTestStore() (*Store, *mockWriter) {
	w := &mockWriter{}
	store := NewWithWriterReader(w, nil)
	return store, w
}

func TestNew(t *testing.T) {
	t.Run("nil writer returns error", func(t *testing.T) {
		_, err := New(Config{})
		if err == nil {
			t.Fatal("expected error for nil writer")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		w := &mockWriter{}
		store, err := New(Config{Writer: w})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if store == nil {
			t.Fatal("expected non-nil store")
		}
	})

	t.Run("default topic", func(t *testing.T) {
		w := &mockWriter{}
		store, _ := New(Config{Writer: w})
		if store.topic != "beluga-workflows" {
			t.Errorf("expected default topic, got %q", store.topic)
		}
	})

	t.Run("custom topic", func(t *testing.T) {
		w := &mockWriter{}
		store, _ := New(Config{Writer: w, Topic: "custom-topic"})
		if store.topic != "custom-topic" {
			t.Errorf("expected custom-topic, got %q", store.topic)
		}
	})
}

func TestSaveAndLoad(t *testing.T) {
	store, _ := newTestStore()
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
	store, _ := newTestStore()

	loaded, err := store.Load(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil for nonexistent workflow")
	}
}

func TestSaveEmptyID(t *testing.T) {
	store, _ := newTestStore()

	err := store.Save(context.Background(), workflow.WorkflowState{})
	if err == nil {
		t.Fatal("expected error for empty workflow ID")
	}
}

func TestSaveOverwrite(t *testing.T) {
	store, _ := newTestStore()
	ctx := context.Background()

	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1", Status: workflow.StatusRunning})
	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1", Status: workflow.StatusCompleted})

	loaded, _ := store.Load(ctx, "wf-1")
	if loaded.Status != workflow.StatusCompleted {
		t.Errorf("expected completed, got %s", loaded.Status)
	}
}

func TestList(t *testing.T) {
	store, _ := newTestStore()
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
	store, _ := newTestStore()

	results, err := store.List(context.Background(), workflow.WorkflowFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestDelete(t *testing.T) {
	store, _ := newTestStore()
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
	store, _ := newTestStore()

	if err := store.Delete(context.Background(), "nonexistent"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestSaveWithHistory(t *testing.T) {
	store, _ := newTestStore()
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
	w := &mockWriter{}
	r := &mockReader{}
	store := NewWithWriterReader(w, r)

	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !w.closed {
		t.Error("expected writer to be closed")
	}
	if !r.closed {
		t.Error("expected reader to be closed")
	}
}

func TestCloseNilReader(t *testing.T) {
	w := &mockWriter{}
	store := NewWithWriterReader(w, nil)

	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestSaveWriteError(t *testing.T) {
	w := &mockWriter{err: fmt.Errorf("write failed")}
	store := NewWithWriterReader(w, nil)

	err := store.Save(context.Background(), workflow.WorkflowState{WorkflowID: "wf-1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInterfaceCompliance(t *testing.T) {
	var _ workflow.WorkflowStore = (*Store)(nil)
}
