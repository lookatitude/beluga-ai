package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// Compile-time check that MindMapPlanner implements Planner.
var _ Planner = (*MindMapPlanner)(nil)

func init() {
	RegisterPlanner("mindmap", func(cfg PlannerConfig) (Planner, error) {
		if cfg.LLM == nil {
			return nil, fmt.Errorf("mindmap planner requires an LLM")
		}
		var opts []MindMapOption
		if maxNodes, ok := cfg.Extra["max_nodes"].(int); ok {
			opts = append(opts, WithMaxNodes(maxNodes))
		}
		if threshold, ok := cfg.Extra["coherence_threshold"].(float64); ok {
			opts = append(opts, WithCoherenceThreshold(threshold))
		}
		return NewMindMapPlanner(cfg.LLM, opts...), nil
	})
}

// MindMapOption configures a MindMapPlanner.
type MindMapOption func(*MindMapPlanner)

// WithMaxNodes sets the maximum number of nodes the planner will add per
// planning iteration.
func WithMaxNodes(n int) MindMapOption {
	return func(p *MindMapPlanner) {
		if n > 0 {
			p.maxNodes = n
		}
	}
}

// WithCoherenceThreshold sets the minimum coherence score required for the
// planner to consider the graph satisfactory. If the coherence falls below
// this threshold during replanning, the planner will attempt to resolve
// contradictions.
func WithCoherenceThreshold(t float64) MindMapOption {
	return func(p *MindMapPlanner) {
		p.coherenceThreshold = clampScore(t)
	}
}

// MindMapPlanner implements the Planner interface using a structured reasoning
// graph (mind map). During Plan and Replan, it constructs a ReasoningGraph of
// claims, evidence, questions, and conclusions connected by support,
// contradiction, derivation, and refinement edges. It uses coherence checking
// between iterations to ensure the reasoning remains consistent.
type MindMapPlanner struct {
	llm                llm.ChatModel
	maxNodes           int
	coherenceThreshold float64

	// graph is persisted across Plan/Replan calls within one agent loop.
	graph *ReasoningGraph
}

// NewMindMapPlanner creates a new MindMapPlanner with the given options.
func NewMindMapPlanner(model llm.ChatModel, opts ...MindMapOption) *MindMapPlanner {
	p := &MindMapPlanner{
		llm:                model,
		maxNodes:           20,
		coherenceThreshold: 0.5,
		graph:              NewReasoningGraph(),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Graph returns the current reasoning graph for inspection.
func (p *MindMapPlanner) Graph() *ReasoningGraph {
	return p.graph
}

// Plan generates actions by constructing a reasoning graph from the input and
// any available observations. It extracts claims, evidence, and questions from
// the LLM, adds them as nodes, and creates edges based on relationships.
func (p *MindMapPlanner) Plan(ctx context.Context, state PlannerState) ([]Action, error) {
	// Reset graph on fresh plan.
	p.graph = NewReasoningGraph()

	if err := p.populateGraph(ctx, state); err != nil {
		return nil, fmt.Errorf("mindmap plan: %w", err)
	}

	return p.synthesize(ctx, state)
}

// Replan updates the reasoning graph with new observations, checks coherence,
// resolves contradictions if needed, and then synthesizes actions.
func (p *MindMapPlanner) Replan(ctx context.Context, state PlannerState) ([]Action, error) {
	// Add observation results as evidence nodes.
	for _, obs := range state.Observations {
		if obs.Result != nil {
			text := extractContentText(obs.Result.Content)
			if text != "" {
				p.graph.AddNode(NodeEvidence, text, 0.7, map[string]any{
					"source": "observation",
				})
			}
		}
	}

	// Add new reasoning nodes from LLM analysis.
	if err := p.populateGraph(ctx, state); err != nil {
		return nil, fmt.Errorf("mindmap replan: %w", err)
	}

	// Coherence check: if below threshold, attempt to resolve contradictions.
	if p.graph.CoherenceScore() < p.coherenceThreshold {
		if err := p.resolveContradictions(ctx, state); err != nil {
			return nil, fmt.Errorf("mindmap resolve contradictions: %w", err)
		}
	}

	return p.synthesize(ctx, state)
}

// populateGraph asks the LLM to analyze the current state and extract
// structured reasoning elements (claims, evidence, questions) to add as graph nodes.
func (p *MindMapPlanner) populateGraph(ctx context.Context, state PlannerState) error {
	graphSummary := p.graphSummary()

	// Spotlight user-supplied input using XML-style delimiters to guard
	// against prompt injection. See .claude/rules/security.md.
	prompt := fmt.Sprintf(
		"Analyze the following problem and produce structured reasoning elements.\n\n"+
			"Problem: <user_input>%s</user_input>\n\n", state.Input)

	if graphSummary != "" {
		prompt += fmt.Sprintf("Existing reasoning context:\n%s\n\n", graphSummary)
	}

	if len(state.Observations) > 0 {
		prompt += "Recent observations:\n"
		for _, obs := range state.Observations {
	if len(state.Observations) > 0 {
		prompt += "Recent observations:\n"
		for _, obs := range state.Observations {
			if obs.Result != nil {
				text := extractContentText(obs.Result.Content)
				if text != "" {
					prompt += fmt.Sprintf("- %s\n", text)
				}
			}
		}
		prompt += "\n"
	}
	}

	prompt += `For each element, output exactly one line in this format:
TYPE|CONTENT|SCORE|RELATES_TO_INDEX|RELATION

Where TYPE is one of: claim, evidence, question, conclusion
SCORE is a confidence from 0.0 to 1.0
RELATES_TO_INDEX is the 1-based index of a previous element in this list (0 if none)
RELATION is one of: supports, contradicts, derives_from, refines (or empty if no relation)

Produce up to 8 elements:`

	resp, err := p.llm.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("Extract structured reasoning elements. Output one per line in the specified format."),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return err
	}

	return p.parseAndAddNodes(resp.Text())
}

// parseAndAddNodes parses the LLM's structured output and adds nodes/edges
// to the graph.
func (p *MindMapPlanner) parseAndAddNodes(text string) error {
	lines := strings.Split(text, "\n")

	// Track IDs for relating back.
		addedIDs = append(addedIDs, id)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 3 {
			continue
		}

		nodeType := parseNodeType(strings.TrimSpace(parts[0]))
		content := strings.TrimSpace(parts[1])
		if content == "" {
			continue
		}

		score := parseFloat(strings.TrimSpace(parts[2]), 0.5)

		if p.maxNodes > 0 && p.graph.NodeCount()+1 > p.maxNodes {
			break
		}

		id := p.graph.AddNode(nodeType, content, score, nil)
		addedIDs = append(addedIDs, id)

		// Parse relationship if present.
		if len(parts) >= 5 {
			relIdx := parseInt(strings.TrimSpace(parts[3]), 0)
			relType := parseEdgeType(strings.TrimSpace(parts[4]))
			if relIdx > 0 && relIdx <= len(addedIDs)-1 && relType != "" {
				targetID := addedIDs[relIdx-1]
				// Ignore edge errors (e.g., duplicate) — best effort.
				_ = p.graph.AddEdge(id, targetID, relType, score)
			}
		}
	}

	return nil
}

// resolveContradictions asks the LLM to evaluate contradictions and add
// conclusion nodes that resolve them.
func (p *MindMapPlanner) resolveContradictions(ctx context.Context, state PlannerState) error {
	contradictions := p.graph.FindContradictions()
	if len(contradictions) == 0 {
		return nil
	}

	var desc strings.Builder
	desc.WriteString("The following contradictions exist in the reasoning:\n\n")
	for i, c := range contradictions {
		fromNode := p.graph.GetNode(c.From)
		toNode := p.graph.GetNode(c.To)
		if fromNode == nil || toNode == nil {
			continue
		}
		fmt.Fprintf(&desc, "%d. \"%s\" contradicts \"%s\"\n", i+1, fromNode.Content, toNode.Content)
	}

	prompt := fmt.Sprintf(
		"Problem: <user_input>%s</user_input>\n\n%s\n"+
			"For each contradiction, provide a resolution as a conclusion. "+
			"Output one conclusion per line:\nCONCLUSION|content|score\n",
		state.Input, desc.String())

	resp, err := p.llm.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("Resolve contradictions in reasoning. Output conclusions."),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return err
	}

	// Parse resolution conclusions.
	for _, line := range strings.Split(resp.Text(), "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 2 {
			continue
		}
		content := strings.TrimSpace(parts[1])
		if content == "" {
			continue
		}
		score := 0.8
		if len(parts) >= 3 {
			score = parseFloat(strings.TrimSpace(parts[2]), 0.8)
		}

		if p.maxNodes > 0 && p.graph.NodeCount()+1 > p.maxNodes {
			break
		}

		p.graph.AddNode(NodeConclusion, content, score, map[string]any{
			"source": "contradiction_resolution",
		})
	}

	return nil
}

// synthesize produces actions from the current state of the reasoning graph.
func (p *MindMapPlanner) synthesize(ctx context.Context, state PlannerState) ([]Action, error) {
	// Build a reasoning summary from the graph.
	var reasoning strings.Builder
	reasoning.WriteString("Structured reasoning (mind map):\n\n")

	for _, n := range p.graph.Nodes() {
		fmt.Fprintf(&reasoning, "[%s] (%.2f) %s\n", n.Type, n.Score, n.Content)
	}

	contradictions := p.graph.FindContradictions()
	if len(contradictions) > 0 {
		fmt.Fprintf(&reasoning, "\nContradictions found: %d\n", len(contradictions))
	}
	fmt.Fprintf(&reasoning, "\nCoherence score: %.2f\n", p.graph.CoherenceScore())

	messages := buildMessagesFromState(state)
	structureMsg := schema.NewSystemMessage(reasoning.String())

	msgs := make([]schema.Message, 0, len(messages)+1)
	msgs = append(msgs, structureMsg)
	msgs = append(msgs, messages...)

	model := p.llm
	if len(state.Tools) > 0 {
		model = model.BindTools(toolDefinitions(state.Tools))
	}

	resp, err := model.Generate(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("mindmap synthesize: %w", err)
	}

	return parseAIResponse(resp), nil
}

// graphSummary returns a textual summary of the current graph state.
func (p *MindMapPlanner) graphSummary() string {
	nodes := p.graph.Nodes()
	if len(nodes) == 0 {
		return ""
	}

	var b strings.Builder
	for _, n := range nodes {
		fmt.Fprintf(&b, "[%s] %s (score=%.2f)\n", n.Type, n.Content, n.Score)
	}
	return b.String()
}

// extractContentText concatenates all text content parts into a single string.
func extractContentText(parts []schema.ContentPart) string {
	var b strings.Builder
	for _, p := range parts {
		if tp, ok := p.(schema.TextPart); ok {
			if b.Len() > 0 {
				b.WriteString(" ")
			}
			b.WriteString(tp.Text)
		}
	}
	return b.String()
}

// parseNodeType converts a string to a NodeType, defaulting to NodeClaim.
func parseNodeType(s string) NodeType {
	switch strings.ToLower(s) {
	case "claim":
		return NodeClaim
	case "evidence":
		return NodeEvidence
	case "question":
		return NodeQuestion
	case "conclusion":
		return NodeConclusion
	default:
		return NodeClaim
	}
}

// parseEdgeType converts a string to an EdgeType, returning empty string if unrecognized.
func parseEdgeType(s string) EdgeType {
	switch strings.ToLower(s) {
	case "supports":
		return EdgeSupports
	case "contradicts":
		return EdgeContradicts
	case "derives_from":
		return EdgeDerivesFrom
	case "refines":
		return EdgeRefines
	default:
		return ""
	}
}

// parseFloat parses a float64 from a string, returning the default on failure.
func parseFloat(s string, def float64) float64 {
	var v float64
	if _, err := fmt.Sscanf(s, "%f", &v); err != nil {
		return def
	}
	return v
}

// parseInt parses an int from a string, returning the default on failure.
func parseInt(s string, def int) int {
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return def
	}
	return v
}
