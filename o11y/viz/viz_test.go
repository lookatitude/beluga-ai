package viz

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestDefaultCollector_RecordAndBuild(t *testing.T) {
	collector := NewCollector(WithGraphID("test-graph"))

	now := time.Now()
	events := []ExecutionEvent{
		{ID: "n1", Type: "agent", Label: "Start", AgentID: "a1", Timestamp: now, Duration: 10 * time.Millisecond, Status: "success"},
		{ID: "n2", ParentID: "n1", Type: "llm", Label: "Generate", AgentID: "a1", Timestamp: now.Add(time.Millisecond), Duration: 5 * time.Millisecond, Status: "success"},
		{ID: "n3", ParentID: "n1", Type: "tool", Label: "Search", AgentID: "a1", Timestamp: now.Add(6 * time.Millisecond), Duration: 3 * time.Millisecond, Status: "error"},
	}

	ctx := context.Background()
	for _, e := range events {
		if err := collector.Record(ctx, e); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	graph, err := collector.Build(ctx)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if graph.ID != "test-graph" {
		t.Errorf("graph ID = %q, want %q", graph.ID, "test-graph")
	}
	if len(graph.Nodes) != 3 {
		t.Errorf("nodes = %d, want 3", len(graph.Nodes))
	}
	if len(graph.Edges) != 2 {
		t.Errorf("edges = %d, want 2", len(graph.Edges))
	}
	if graph.RootNodeID != "n1" {
		t.Errorf("root = %q, want %q", graph.RootNodeID, "n1")
	}
}

func TestDefaultCollector_EmptyBuild(t *testing.T) {
	collector := NewCollector()
	graph, err := collector.Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(graph.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(graph.Nodes))
	}
}

func TestJSONRenderer(t *testing.T) {
	graph := &ExecutionGraph{
		ID:         "g1",
		RootNodeID: "n1",
		Nodes:      []Node{{ID: "n1", Label: "test", Type: "agent", Status: "success"}},
		Edges:      nil,
	}

	renderer := &JSONRenderer{}
	data, err := renderer.Render(context.Background(), graph)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	s := string(data)
	if !strings.Contains(s, `"id": "g1"`) {
		t.Errorf("JSON should contain graph ID")
	}
	if !strings.Contains(s, `"n1"`) {
		t.Errorf("JSON should contain node ID")
	}
}

func TestDOTRenderer(t *testing.T) {
	graph := &ExecutionGraph{
		ID:         "g1",
		RootNodeID: "n1",
		Nodes: []Node{
			{ID: "n1", Label: "Start", Type: "agent", Status: "success", Duration: 10 * time.Millisecond},
			{ID: "n2", Label: "LLM Call", Type: "llm", Status: "error", Duration: 5 * time.Millisecond},
		},
		Edges: []Edge{
			{From: "n1", To: "n2", Label: "calls"},
		},
	}

	renderer := &DOTRenderer{}
	data, err := renderer.Render(context.Background(), graph)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	s := string(data)
	if !strings.Contains(s, "digraph execution") {
		t.Error("DOT should start with digraph")
	}
	if !strings.Contains(s, `"n1"`) {
		t.Error("DOT should contain node n1")
	}
	if !strings.Contains(s, `"n1" -> "n2"`) {
		t.Error("DOT should contain edge from n1 to n2")
	}
	if !strings.Contains(s, "green") {
		t.Error("DOT should contain green for success nodes")
	}
	if !strings.Contains(s, "red") {
		t.Error("DOT should contain red for error nodes")
	}
}

func TestRendererRegistry(t *testing.T) {
	names := ListRenderers()
	if len(names) < 2 {
		t.Errorf("expected at least 2 renderers, got %d", len(names))
	}

	r, err := NewRenderer("json")
	if err != nil {
		t.Fatalf("NewRenderer(json): %v", err)
	}
	if r == nil {
		t.Error("expected non-nil renderer")
	}

	r, err = NewRenderer("dot")
	if err != nil {
		t.Fatalf("NewRenderer(dot): %v", err)
	}
	if r == nil {
		t.Error("expected non-nil renderer")
	}

	_, err = NewRenderer("nonexistent")
	if err == nil {
		t.Error("expected error for unknown renderer")
	}
}
