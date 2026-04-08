package agent

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// NodeType identifies the kind of node in a reasoning graph.
type NodeType string

const (
	// NodeClaim is an assertion or proposition in the reasoning.
	NodeClaim NodeType = "claim"
	// NodeEvidence is supporting data or observation.
	NodeEvidence NodeType = "evidence"
	// NodeQuestion is an open question or uncertainty.
	NodeQuestion NodeType = "question"
	// NodeConclusion is a derived conclusion from reasoning.
	NodeConclusion NodeType = "conclusion"
)

// EdgeType identifies the relationship between two nodes.
type EdgeType string

const (
	// EdgeSupports indicates the source supports the target.
	EdgeSupports EdgeType = "supports"
	// EdgeContradicts indicates the source contradicts the target.
	EdgeContradicts EdgeType = "contradicts"
	// EdgeDerivesFrom indicates the target is derived from the source.
	EdgeDerivesFrom EdgeType = "derives_from"
	// EdgeRefines indicates the source refines or improves the target.
	EdgeRefines EdgeType = "refines"
)

// GraphNode represents a single node in the reasoning graph.
type GraphNode struct {
	// ID uniquely identifies this node.
	ID string
	// Type is the kind of reasoning element.
	Type NodeType
	// Content is the textual content of this node.
	Content string
	// Score is a confidence or relevance score (0.0-1.0).
	Score float64
	// Metadata holds node-specific data.
	Metadata map[string]any
}

// GraphEdge represents a directed relationship between two nodes.
type GraphEdge struct {
	// From is the source node ID.
	From string
	// To is the target node ID.
	To string
	// Type identifies the relationship.
	Type EdgeType
	// Weight is the strength of the relationship (0.0-1.0).
	Weight float64
}

// ReasoningGraph is a thread-safe directed graph of reasoning nodes and edges.
// It supports claims, evidence, questions, and conclusions connected by
// support, contradiction, derivation, and refinement relationships.
type ReasoningGraph struct {
	nodes  map[string]*GraphNode
	edges  []GraphEdge
	nextID atomic.Int64
	mu     sync.RWMutex
}

// NewReasoningGraph creates a new empty reasoning graph.
func NewReasoningGraph() *ReasoningGraph {
	return &ReasoningGraph{
		nodes: make(map[string]*GraphNode),
	}
}

// AddNode adds a node to the graph and returns its assigned ID.
// The provided node's ID field is overwritten with a generated unique ID.
func (g *ReasoningGraph) AddNode(nodeType NodeType, content string, score float64, metadata map[string]any) string {
	id := fmt.Sprintf("rg_%d", g.nextID.Add(1))

	node := &GraphNode{
		ID:       id,
		Type:     nodeType,
		Content:  content,
		Score:    clampScore(score),
		Metadata: metadata,
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes[id] = node
	return id
}

// AddEdge adds a directed edge between two nodes. Returns an error if either
// node does not exist in the graph.
func (g *ReasoningGraph) AddEdge(from, to string, edgeType EdgeType, weight float64) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.nodes[from]; !ok {
		return fmt.Errorf("source node %q not found", from)
	}
	if _, ok := g.nodes[to]; !ok {
		return fmt.Errorf("target node %q not found", to)
	}

	g.edges = append(g.edges, GraphEdge{
		From:   from,
		To:     to,
		Type:   edgeType,
		Weight: clampScore(weight),
	})
	return nil
}

// GetNode returns the node with the given ID, or nil if not found.
func (g *ReasoningGraph) GetNode(id string) *GraphNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.nodes[id]
}

// NodeCount returns the number of nodes in the graph.
func (g *ReasoningGraph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// EdgeCount returns the number of edges in the graph.
func (g *ReasoningGraph) EdgeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.edges)
}

// Nodes returns a snapshot of all nodes in the graph, sorted by ID.
func (g *ReasoningGraph) Nodes() []*GraphNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make([]*GraphNode, 0, len(g.nodes))
	for _, n := range g.nodes {
		result = append(result, n)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

// Edges returns a snapshot of all edges in the graph.
func (g *ReasoningGraph) Edges() []GraphEdge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make([]GraphEdge, len(g.edges))
	copy(result, g.edges)
	return result
}

// Neighbors returns all nodes directly connected to the given node (both
// outgoing and incoming edges), along with the connecting edges.
func (g *ReasoningGraph) Neighbors(nodeID string) ([]*GraphNode, []GraphEdge) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, ok := g.nodes[nodeID]; !ok {
		return nil, nil
	}

	seen := make(map[string]bool)
	var neighbors []*GraphNode
	var relEdges []GraphEdge

	for _, e := range g.edges {
		var neighborID string
		if e.From == nodeID {
			neighborID = e.To
		} else if e.To == nodeID {
			neighborID = e.From
		} else {
			continue
		}

		relEdges = append(relEdges, e)
		if !seen[neighborID] {
			seen[neighborID] = true
			if n, ok := g.nodes[neighborID]; ok {
				neighbors = append(neighbors, n)
			}
		}
	}

	return neighbors, relEdges
}

// FindContradictions returns all pairs of edges where one supports and the
// other contradicts the same target, or any EdgeContradicts edges in the graph.
func (g *ReasoningGraph) FindContradictions() []GraphEdge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var contradictions []GraphEdge
	for _, e := range g.edges {
		if e.Type == EdgeContradicts {
			contradictions = append(contradictions, e)
		}
	}
	return contradictions
}

// CoherenceScore computes an overall coherence score for the graph.
// The score is in [0.0, 1.0] where 1.0 means fully coherent.
//
// The algorithm considers:
//   - Average node confidence scores
//   - Ratio of supporting vs contradicting edges
//   - Connectivity (isolated nodes lower coherence)
//
// An empty graph returns 1.0 (vacuously coherent).
func (g *ReasoningGraph) CoherenceScore() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.nodes) == 0 {
		return 1.0
	}

	// Component 1: Average node score (weighted by 0.4)
	var nodeScoreSum float64
	for _, n := range g.nodes {
		nodeScoreSum += n.Score
	}
	avgNodeScore := nodeScoreSum / float64(len(g.nodes))

	// Component 2: Support vs contradiction ratio (weighted by 0.4)
	var supportCount, contradictCount int
	for _, e := range g.edges {
		switch e.Type {
		case EdgeSupports:
			supportCount++
		case EdgeContradicts:
			contradictCount++
		}
	}

	var edgeCoherence float64
	totalRelational := supportCount + contradictCount
	if totalRelational == 0 {
		edgeCoherence = 1.0 // no contradictions = coherent
	} else {
		edgeCoherence = float64(supportCount) / float64(totalRelational)
	}

	// Component 3: Connectivity ratio (weighted by 0.2)
	// Nodes that participate in at least one edge vs total nodes.
	connected := make(map[string]bool)
	for _, e := range g.edges {
		connected[e.From] = true
		connected[e.To] = true
	}
	var connectivityRatio float64
	if len(g.nodes) <= 1 {
		connectivityRatio = 1.0
	} else {
		connectivityRatio = float64(len(connected)) / float64(len(g.nodes))
	}

	score := 0.4*avgNodeScore + 0.4*edgeCoherence + 0.2*connectivityRatio
	return clampScore(score)
}

// reasoningGraphExport is the JSON-serializable representation of a ReasoningGraph.
type reasoningGraphExport struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// ExportJSON returns the graph as a JSON byte slice.
func (g *ReasoningGraph) ExportJSON() ([]byte, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	export := reasoningGraphExport{
		Nodes: make([]GraphNode, 0, len(g.nodes)),
		Edges: make([]GraphEdge, len(g.edges)),
	}

	// Sort nodes by ID for deterministic output.
	ids := make([]string, 0, len(g.nodes))
	for id := range g.nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		export.Nodes = append(export.Nodes, *g.nodes[id])
	}

	copy(export.Edges, g.edges)

	return json.Marshal(export)
}

// ExportDOT returns the graph in Graphviz DOT format.
func (g *ReasoningGraph) ExportDOT() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var b strings.Builder
	b.WriteString("digraph reasoning {\n")
	b.WriteString("  rankdir=TB;\n")
	b.WriteString("  node [shape=box, style=filled];\n\n")

	// Sort nodes for deterministic output.
	ids := make([]string, 0, len(g.nodes))
	for id := range g.nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		n := g.nodes[id]
		color := nodeColor(n.Type)
		label := dotEscape(n.Content)
		if len(label) > 60 {
			label = label[:57] + "..."
		}
		fmt.Fprintf(&b, "  %q [label=%q, fillcolor=%q, tooltip=\"score=%.2f\"];\n",
			n.ID, label, color, n.Score)
	}

	b.WriteString("\n")

	for _, e := range g.edges {
		style := edgeStyle(e.Type)
		fmt.Fprintf(&b, "  %q -> %q [label=%q, style=%q, penwidth=%.1f];\n",
			e.From, e.To, string(e.Type), style, e.Weight*3)
	}

	b.WriteString("}\n")
	return b.String()
}

// clampScore clamps a value to the range [0.0, 1.0].
func clampScore(v float64) float64 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// nodeColor returns a DOT fill color for the given node type.
func nodeColor(nt NodeType) string {
	switch nt {
	case NodeClaim:
		return "#AED6F1"
	case NodeEvidence:
		return "#A9DFBF"
	case NodeQuestion:
		return "#F9E79F"
	case NodeConclusion:
		return "#D7BDE2"
	default:
		return "#F2F3F4"
	}
}

// edgeStyle returns a DOT line style for the given edge type.
func edgeStyle(et EdgeType) string {
	switch et {
	case EdgeContradicts:
		return "dashed"
	case EdgeRefines:
		return "dotted"
	default:
		return "solid"
	}
}

// dotEscape escapes a string for use in a DOT label.
func dotEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
