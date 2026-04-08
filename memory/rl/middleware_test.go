package rl

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
)

// mockMemory is a test double for memory.Memory.
type mockMemory struct {
	saveCount   atomic.Int64
	loadCount   atomic.Int64
	searchCount atomic.Int64
	clearCount  atomic.Int64
	saveErr     error
	loadMsgs    []schema.Message
	searchDocs  []schema.Document
}

func (m *mockMemory) Save(_ context.Context, _, _ schema.Message) error {
	m.saveCount.Add(1)
	return m.saveErr
}

func (m *mockMemory) Load(_ context.Context, _ string) ([]schema.Message, error) {
	m.loadCount.Add(1)
	return m.loadMsgs, nil
}

func (m *mockMemory) Search(_ context.Context, _ string, _ int) ([]schema.Document, error) {
	m.searchCount.Add(1)
	return m.searchDocs, nil
}

func (m *mockMemory) Clear(_ context.Context) error {
	m.clearCount.Add(1)
	return nil
}

// mockPolicy is a test double for MemoryPolicy.
type mockPolicy struct {
	action     MemoryAction
	confidence float64
	err        error
}

func (p *mockPolicy) Decide(_ context.Context, _ PolicyFeatures) (MemoryAction, float64, error) {
	return p.action, p.confidence, p.err
}

// mockExtractor is a test double for FeatureExtractor.
type mockExtractor struct {
	features PolicyFeatures
	err      error
}

func (e *mockExtractor) Extract(_ context.Context, _ memory.Memory, _, _ schema.Message) (PolicyFeatures, error) {
	return e.features, e.err
}

func newInput() schema.Message  { return schema.NewHumanMessage("hello") }
func newOutput() schema.Message { return schema.NewAIMessage("world") }

func TestPolicyMemory_Save_ActionAdd(t *testing.T) {
	mem := &mockMemory{}
	policy := &mockPolicy{action: ActionAdd, confidence: 0.9}

	pm := New(mem, WithPolicy(policy))
	err := pm.Save(context.Background(), newInput(), newOutput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mem.saveCount.Load() != 1 {
		t.Errorf("save count = %d, want 1", mem.saveCount.Load())
	}
}

func TestPolicyMemory_Save_ActionNoop(t *testing.T) {
	mem := &mockMemory{}
	policy := &mockPolicy{action: ActionNoop, confidence: 0.8}

	pm := New(mem, WithPolicy(policy))
	err := pm.Save(context.Background(), newInput(), newOutput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mem.saveCount.Load() != 0 {
		t.Errorf("save count = %d, want 0 (noop)", mem.saveCount.Load())
	}
}

func TestPolicyMemory_Save_ActionDelete(t *testing.T) {
	mem := &mockMemory{}
	policy := &mockPolicy{action: ActionDelete, confidence: 0.7}

	pm := New(mem, WithPolicy(policy))
	err := pm.Save(context.Background(), newInput(), newOutput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mem.saveCount.Load() != 0 {
		t.Errorf("save count = %d, want 0 (delete does not save)", mem.saveCount.Load())
	}
}

func TestPolicyMemory_Save_ActionUpdate(t *testing.T) {
	mem := &mockMemory{}
	policy := &mockPolicy{action: ActionUpdate, confidence: 0.8}

	pm := New(mem, WithPolicy(policy))
	err := pm.Save(context.Background(), newInput(), newOutput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Update delegates to Save.
	if mem.saveCount.Load() != 1 {
		t.Errorf("save count = %d, want 1", mem.saveCount.Load())
	}
}

func TestPolicyMemory_Save_PolicyError(t *testing.T) {
	mem := &mockMemory{}
	policy := &mockPolicy{err: errors.New("policy failed")}

	pm := New(mem, WithPolicy(policy))
	err := pm.Save(context.Background(), newInput(), newOutput())
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "policy failed" {
		t.Errorf("error = %q, want %q", err.Error(), "policy failed")
	}
}

func TestPolicyMemory_Save_WithHooksOnDecision(t *testing.T) {
	mem := &mockMemory{}
	policy := &mockPolicy{action: ActionAdd, confidence: 0.9}

	var hookAction MemoryAction
	var hookConf float64
	hooks := Hooks{
		OnDecision: func(_ context.Context, _ PolicyFeatures, action MemoryAction, conf float64) error {
			hookAction = action
			hookConf = conf
			return nil
		},
	}

	pm := New(mem, WithPolicy(policy), WithHooks(hooks))
	err := pm.Save(context.Background(), newInput(), newOutput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hookAction != ActionAdd {
		t.Errorf("hook action = %v, want ActionAdd", hookAction)
	}
	if hookConf != 0.9 {
		t.Errorf("hook confidence = %v, want 0.9", hookConf)
	}
}

func TestPolicyMemory_Save_HookError(t *testing.T) {
	mem := &mockMemory{}
	policy := &mockPolicy{action: ActionAdd, confidence: 0.9}

	hooks := Hooks{
		OnDecision: func(_ context.Context, _ PolicyFeatures, _ MemoryAction, _ float64) error {
			return errors.New("hook rejected")
		},
	}

	pm := New(mem, WithPolicy(policy), WithHooks(hooks))
	err := pm.Save(context.Background(), newInput(), newOutput())
	if err == nil {
		t.Fatal("expected error from hook")
	}

	// Should not have saved.
	if mem.saveCount.Load() != 0 {
		t.Errorf("save count = %d, want 0", mem.saveCount.Load())
	}
}

func TestPolicyMemory_Save_WithCollector(t *testing.T) {
	mem := &mockMemory{}
	policy := &mockPolicy{action: ActionAdd, confidence: 0.9}
	collector := NewTrajectoryCollector()

	pm := New(mem, WithPolicy(policy), WithCollector(collector))
	err := pm.Save(context.Background(), newInput(), newOutput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// End the episode.
	if err := collector.EndEpisode(context.Background(), true); err != nil {
		t.Fatal(err)
	}

	episodes := collector.Episodes()
	if len(episodes) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(episodes))
	}
	if len(episodes[0].Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(episodes[0].Steps))
	}
	if episodes[0].Steps[0].Action != ActionAdd {
		t.Errorf("recorded action = %v, want ActionAdd", episodes[0].Steps[0].Action)
	}
}

func TestPolicyMemory_Load(t *testing.T) {
	expected := []schema.Message{newInput()}
	mem := &mockMemory{loadMsgs: expected}

	pm := New(mem)
	msgs, err := pm.Load(context.Background(), "test query")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("msgs len = %d, want 1", len(msgs))
	}
}

func TestPolicyMemory_Search(t *testing.T) {
	expected := []schema.Document{{ID: "doc1", Content: "test"}}
	mem := &mockMemory{searchDocs: expected}

	pm := New(mem)
	docs, err := pm.Search(context.Background(), "query", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("docs len = %d, want 1", len(docs))
	}
}

func TestPolicyMemory_Clear(t *testing.T) {
	mem := &mockMemory{}
	pm := New(mem)

	err := pm.Clear(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mem.clearCount.Load() != 1 {
		t.Errorf("clear count = %d, want 1", mem.clearCount.Load())
	}
}

func TestPolicyMemory_DefaultPolicy(t *testing.T) {
	mem := &mockMemory{}
	pm := New(mem) // No WithPolicy — should use HeuristicPolicy.

	// Should not panic.
	err := pm.Save(context.Background(), newInput(), newOutput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
