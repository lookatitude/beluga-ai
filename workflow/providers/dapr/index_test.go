package dapr

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/workflow"
)

// TestStore_ListSurvivesRestart is a regression test for FWC-12: List only
// knew the IDs saved by the current process.
func TestStore_ListSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	client := newMockStateClient()
	first := NewWithClient(client, "")
	for _, id := range []string{"a", "b", "c"} {
		if err := first.Save(ctx, workflow.WorkflowState{WorkflowID: id, Status: workflow.StatusCompleted}); err != nil {
			t.Fatal(err)
		}
	}
	// Saving an existing ID again must not duplicate it.
	if err := first.Save(ctx, workflow.WorkflowState{WorkflowID: "a", Status: workflow.StatusRunning}); err != nil {
		t.Fatal(err)
	}
	if err := first.Delete(ctx, "c"); err != nil {
		t.Fatal(err)
	}

	second := NewWithClient(client, "") // "restart": empty local index
	got, err := second.List(ctx, workflow.WorkflowFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].WorkflowID != "a" || got[1].WorkflowID != "b" {
		t.Fatalf("List after restart = %+v, want a and b", got)
	}
	if err := second.Delete(ctx, "zzz"); err != nil {
		t.Fatalf("deleting an unindexed id: %v", err)
	}
}

func TestStore_IndexErrors(t *testing.T) {
	ctx := context.Background()
	client := newMockStateClient()
	s := NewWithClient(client, "")
	if err := s.Save(ctx, workflow.WorkflowState{WorkflowID: IndexKey}); core.CodeOf(err, "") != core.ErrInvalidInput {
		t.Errorf("reserved key: err = %v", err)
	}
	client.store[IndexKey] = []byte("{bad")
	if _, err := s.List(ctx, workflow.WorkflowFilter{}); err == nil {
		t.Error("expected index decode error")
	}
	if err := s.Save(ctx, workflow.WorkflowState{WorkflowID: "x"}); err == nil {
		t.Error("expected index decode error on save")
	}
}
