package agent

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

func TestReasoningGraph_AddNode(t *testing.T) {
	tests := []struct {
		name     string
		nodeType NodeType
		content  string
		score    float64
		wantType NodeType
	}{
		{name: "claim node", nodeType: NodeClaim, content: "The sky is blue", score: 0.9, wantType: NodeClaim},
		{name: "evidence node", nodeType: NodeEvidence, content: "Observation data", score: 0.8, wantType: NodeEvidence},
		{name: "question node", nodeType: NodeQuestion, content: "Why?", score: 0.5, wantType: NodeQuestion},
		{name: "conclusion node", nodeType: NodeConclusion, content: "Therefore...", score: 1.0, wantType: NodeConclusion},
		{name: "score clamped high", nodeType: NodeClaim, content: "test", score: 1.5, wantType: NodeClaim},
		{name: "score clamped low", nodeType: NodeClaim, content: "test", score: -0.5, wantType: NodeClaim},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewReasoningGraph()
			id := g.AddNode(tt.nodeType, tt.content, tt.score, nil)
			assertNodeProperties(t, g, id, tt.content, tt.wantType)
		})
	}
}

// assertNodeProperties verifies that the node with the given ID exists in the
// graph and has the expected type, content, and a clamped score in [0, 1].
func assertNodeProperties(t *testing.T, g *ReasoningGraph, id, wantContent string, wantType NodeType) {
	t.Helper()
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	node := g.GetNode(id)
	if node == nil {
		t.Fatal("expected node to be retrievable")
	}
	if node.Type != wantType {
		t.Errorf("got type %q, want %q", node.Type, wantType)
	}
	if node.Content != wantContent {
		t.Errorf("got content %q, want %q", node.Content, wantContent)
	}
	if node.Score < 0 || node.Score > 1 {
		t.Errorf("score %f out of [0,1] range", node.Score)
	}
}

func TestReasoningGraph_AddEdge(t *testing.T) {
	g := NewReasoningGraph()
	id1 := g.AddNode(NodeClaim, "claim1", 0.8, nil)
	id2 := g.AddNode(NodeEvidence, "evidence1", 0.7, nil)

	tests := []struct {
		name    string
		from    string
		to      string
		edgeT   EdgeType
		weight  float64
		wantErr bool
	}{
		{name: "valid edge", from: id1, to: id2, edgeT: EdgeSupports, weight: 0.9, wantErr: false},
		{name: "missing source", from: "nonexistent", to: id2, edgeT: EdgeSupports, weight: 0.5, wantErr: true},
		{name: "missing target", from: id1, to: "nonexistent", edgeT: EdgeSupports, weight: 0.5, wantErr: true},
		{name: "contradicts edge", from: id2, to: id1, edgeT: EdgeContradicts, weight: 0.6, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.AddEdge(tt.from, tt.to, tt.edgeT, tt.weight)
			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReasoningGraph_Neighbors(t *testing.T) {
	g := NewReasoningGraph()
	id1 := g.AddNode(NodeClaim, "center", 0.8, nil)
	id2 := g.AddNode(NodeEvidence, "neighbor1", 0.7, nil)
	id3 := g.AddNode(NodeEvidence, "neighbor2", 0.6, nil)
	_ = g.AddNode(NodeClaim, "isolated", 0.5, nil)

	_ = g.AddEdge(id1, id2, EdgeSupports, 0.8)
	_ = g.AddEdge(id3, id1, EdgeDerivesFrom, 0.7)

	neighbors, edges := g.Neighbors(id1)
	if len(neighbors) != 2 {
		t.Errorf("expected 2 neighbors, got %d", len(neighbors))
	}
	if len(edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(edges))
	}

	// Nonexistent node.
	n, e := g.Neighbors("nonexistent")
	if n != nil || e != nil {
		t.Error("expected nil for nonexistent node")
	}
}

func TestReasoningGraph_FindContradictions(t *testing.T) {
	g := NewReasoningGraph()
	id1 := g.AddNode(NodeClaim, "A is true", 0.8, nil)
	id2 := g.AddNode(NodeClaim, "A is false", 0.7, nil)
	id3 := g.AddNode(NodeEvidence, "supports A", 0.9, nil)

	_ = g.AddEdge(id1, id2, EdgeContradicts, 0.9)
	_ = g.AddEdge(id3, id1, EdgeSupports, 0.8)

	contradictions := g.FindContradictions()
	if len(contradictions) != 1 {
		t.Errorf("expected 1 contradiction, got %d", len(contradictions))
	}
	if len(contradictions) > 0 && contradictions[0].Type != EdgeContradicts {
		t.Errorf("expected EdgeContradicts, got %s", contradictions[0].Type)
	}
}

func TestReasoningGraph_FindContradictions_Empty(t *testing.T) {
	g := NewReasoningGraph()
	g.AddNode(NodeClaim, "solo", 0.5, nil)
	contradictions := g.FindContradictions()
	if len(contradictions) != 0 {
		t.Errorf("expected 0 contradictions, got %d", len(contradictions))
	}
}

func TestReasoningGraph_CoherenceScore(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *ReasoningGraph
		wantMin float64
		wantMax float64
	}{
		{
			name:    "empty graph",
			setup:   func() *ReasoningGraph { return NewReasoningGraph() },
			wantMin: 1.0,
			wantMax: 1.0,
		},
		{
			name: "single node",
			setup: func() *ReasoningGraph {
				g := NewReasoningGraph()
				g.AddNode(NodeClaim, "test", 0.8, nil)
				return g
			},
			wantMin: 0.4, // 0.4*0.8 + 0.4*1.0 + 0.2*1.0 = 0.92 but single node no edges
			wantMax: 1.0,
		},
		{
			name: "all supporting",
			setup: func() *ReasoningGraph {
				g := NewReasoningGraph()
				id1 := g.AddNode(NodeClaim, "A", 0.9, nil)
				id2 := g.AddNode(NodeEvidence, "B", 0.8, nil)
				_ = g.AddEdge(id2, id1, EdgeSupports, 0.9)
				return g
			},
			wantMin: 0.8,
			wantMax: 1.0,
		},
		{
			name: "many contradictions",
			setup: func() *ReasoningGraph {
				g := NewReasoningGraph()
				id1 := g.AddNode(NodeClaim, "A", 0.5, nil)
				id2 := g.AddNode(NodeClaim, "B", 0.5, nil)
				id3 := g.AddNode(NodeClaim, "C", 0.5, nil)
				_ = g.AddEdge(id1, id2, EdgeContradicts, 0.9)
				_ = g.AddEdge(id2, id3, EdgeContradicts, 0.8)
				return g
			},
			wantMin: 0.0,
			wantMax: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.setup()
			score := g.CoherenceScore()
			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("coherence score %.4f not in [%.2f, %.2f]", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestReasoningGraph_ExportJSON(t *testing.T) {
	g := NewReasoningGraph()
	id1 := g.AddNode(NodeClaim, "claim", 0.8, nil)
	id2 := g.AddNode(NodeEvidence, "evidence", 0.7, nil)
	_ = g.AddEdge(id1, id2, EdgeSupports, 0.9)

	data, err := g.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON error: %v", err)
	}

	var export struct {
		Nodes []GraphNode `json:"nodes"`
		Edges []GraphEdge `json:"edges"`
	}
	if err := json.Unmarshal(data, &export); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if len(export.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(export.Nodes))
	}
	if len(export.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(export.Edges))
	}
}

func TestReasoningGraph_ExportDOT(t *testing.T) {
	g := NewReasoningGraph()
	id1 := g.AddNode(NodeClaim, "The sky is blue", 0.9, nil)
	id2 := g.AddNode(NodeEvidence, "Rayleigh scattering", 0.8, nil)
	_ = g.AddEdge(id2, id1, EdgeSupports, 0.9)

	dot := g.ExportDOT()

	if !strings.Contains(dot, "digraph reasoning") {
		t.Error("DOT output missing digraph header")
	}
	if !strings.Contains(dot, "The sky is blue") {
		t.Error("DOT output missing node content")
	}
	if !strings.Contains(dot, "supports") {
		t.Error("DOT output missing edge label")
	}
	if !strings.HasSuffix(strings.TrimSpace(dot), "}") {
		t.Error("DOT output missing closing brace")
	}
}

func TestReasoningGraph_ExportDOT_LongContent(t *testing.T) {
	g := NewReasoningGraph()
	longContent := strings.Repeat("a", 100)
	g.AddNode(NodeClaim, longContent, 0.5, nil)

	dot := g.ExportDOT()
	// Label should be truncated.
	if strings.Contains(dot, longContent) {
		t.Error("DOT output should truncate long content")
	}
	if !strings.Contains(dot, "...") {
		t.Error("DOT output should contain ellipsis for truncated content")
	}
}

func TestReasoningGraph_NodeCount_EdgeCount(t *testing.T) {
	g := NewReasoningGraph()
	if g.NodeCount() != 0 {
		t.Errorf("expected 0 nodes, got %d", g.NodeCount())
	}
	if g.EdgeCount() != 0 {
		t.Errorf("expected 0 edges, got %d", g.EdgeCount())
	}

	id1 := g.AddNode(NodeClaim, "a", 0.5, nil)
	id2 := g.AddNode(NodeClaim, "b", 0.5, nil)
	_ = g.AddEdge(id1, id2, EdgeSupports, 0.5)

	if g.NodeCount() != 2 {
		t.Errorf("expected 2 nodes, got %d", g.NodeCount())
	}
	if g.EdgeCount() != 1 {
		t.Errorf("expected 1 edge, got %d", g.EdgeCount())
	}
}

func TestReasoningGraph_GetNode_NotFound(t *testing.T) {
	g := NewReasoningGraph()
	if g.GetNode("nonexistent") != nil {
		t.Error("expected nil for nonexistent node")
	}
}

func TestReasoningGraph_ThreadSafety(t *testing.T) {
	g := NewReasoningGraph()
	const goroutines = 50
	const opsPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				id := g.AddNode(NodeClaim, "concurrent", 0.5, nil)
				_ = g.GetNode(id)
				_ = g.NodeCount()
				_ = g.Nodes()
				_ = g.CoherenceScore()
				_ = g.FindContradictions()
				_ = g.ExportDOT()
			}
		}()
	}

	wg.Wait()

	if g.NodeCount() != goroutines*opsPerGoroutine {
		t.Errorf("expected %d nodes, got %d", goroutines*opsPerGoroutine, g.NodeCount())
	}
}

func TestReasoningGraph_ThreadSafety_Edges(t *testing.T) {
	g := NewReasoningGraph()

	// Pre-create nodes so edge adds can succeed.
	ids := make([]string, 100)
	for i := range ids {
		ids[i] = g.AddNode(NodeClaim, "node", 0.5, nil)
	}

	var wg sync.WaitGroup
	wg.Add(50)
	for i := 0; i < 50; i++ {
		go func(idx int) {
			defer wg.Done()
			from := ids[idx%len(ids)]
			to := ids[(idx+1)%len(ids)]
			_ = g.AddEdge(from, to, EdgeSupports, 0.5)
			_, _ = g.Neighbors(from)
			_ = g.Edges()
		}(i)
	}
	wg.Wait()

	if g.EdgeCount() != 50 {
		t.Errorf("expected 50 edges, got %d", g.EdgeCount())
	}
}

func TestClampScore(t *testing.T) {
	tests := []struct {
		input float64
		want  float64
	}{
		{-1.0, 0.0},
		{0.0, 0.0},
		{0.5, 0.5},
		{1.0, 1.0},
		{1.5, 1.0},
	}
	for _, tt := range tests {
		got := clampScore(tt.input)
		if got != tt.want {
			t.Errorf("clampScore(%f) = %f, want %f", tt.input, got, tt.want)
		}
	}
}
