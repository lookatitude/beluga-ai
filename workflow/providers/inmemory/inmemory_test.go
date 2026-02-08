package inmemory

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/workflow"
)

func TestStore_SaveAndLoad(t *testing.T) {
	store := New()
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

func TestStore_LoadNotFound(t *testing.T) {
	store := New()
	loaded, err := store.Load(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil for nonexistent workflow")
	}
}

func TestStore_SaveEmptyID(t *testing.T) {
	store := New()
	err := store.Save(context.Background(), workflow.WorkflowState{})
	if err == nil {
		t.Fatal("expected error for empty workflow ID")
	}
}

func TestStore_SaveOverwrite(t *testing.T) {
	store := New()
	ctx := context.Background()

	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1", Status: workflow.StatusRunning})
	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1", Status: workflow.StatusCompleted})

	loaded, _ := store.Load(ctx, "wf-1")
	if loaded.Status != workflow.StatusCompleted {
		t.Errorf("expected completed, got %s", loaded.Status)
	}
}

func TestStore_List(t *testing.T) {
	store := New()
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

func TestStore_Delete(t *testing.T) {
	store := New()
	ctx := context.Background()

	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1"})
	store.Delete(ctx, "wf-1")

	loaded, _ := store.Load(ctx, "wf-1")
	if loaded != nil {
		t.Error("expected nil after delete")
	}
}

func TestStore_DeleteNonexistent(t *testing.T) {
	store := New()
	// Should not error.
	if err := store.Delete(context.Background(), "nonexistent"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
