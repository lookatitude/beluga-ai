package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	RegisterPlanner("graph-of-thought", func(cfg PlannerConfig) (Planner, error) {
		if cfg.LLM == nil {
			return nil, fmt.Errorf("graph-of-thought planner requires an LLM")
		}
		var opts []GoTOption
		if ctrl, ok := cfg.Extra["controller"].(Controller); ok {
			opts = append(opts, WithController(ctrl))
		}
		if maxOps, ok := cfg.Extra["max_operations"].(int); ok {
			opts = append(opts, WithMaxOperations(maxOps))
		}
		return NewGoTPlanner(cfg.LLM, opts...), nil
	})
}

// OperationType identifies the kind of graph transformation operation.
type OperationType string

const (
	// OpGenerate creates new thought nodes from a source node.
	OpGenerate OperationType = "generate"
	// OpMerge combines multiple thought nodes into a single synthesized node.
	OpMerge OperationType = "merge"
	// OpSplit divides a thought node into multiple sub-thoughts.
	OpSplit OperationType = "split"
	// OpLoop re-evaluates and refines a thought node.
	OpLoop OperationType = "loop"
	// OpAggregate collects all leaf nodes and produces a final synthesis.
	OpAggregate OperationType = "aggregate"
)

// Operation describes a graph transformation to apply to the thought graph.
type Operation struct {
	// Type identifies the kind of operation.
	Type OperationType
	// NodeIDs lists the IDs of nodes this operation acts upon.
	NodeIDs []string
	// Args holds operation-specific parameters.
	Args map[string]any
}

// ThoughtNode represents a single node in the thought graph.
type ThoughtNode struct {
	// ID uniquely identifies this thought node.
	ID string
	// Content is the text content of this thought.
	Content string
	// Score is the evaluated quality of this thought (0.0-1.0).
	Score float64
	// Children holds the IDs of child nodes.
	Children []string
	// Parents holds the IDs of parent nodes.
	Parents []string
}

// ThoughtGraph is a directed graph of thought nodes used for
// Graph of Thought reasoning.
type ThoughtGraph struct {
	// Nodes maps node IDs to thought nodes.
	Nodes map[string]*ThoughtNode
	// mu protects concurrent access.
	mu sync.RWMutex
	// nextID is an atomic counter for generating unique node IDs.
	nextID atomic.Int64
}

// NewThoughtGraph creates a new empty thought graph.
func NewThoughtGraph() *ThoughtGraph {
	return &ThoughtGraph{
		Nodes: make(map[string]*ThoughtNode),
	}
}

// AddNode adds a node to the graph and returns its assigned ID.
func (g *ThoughtGraph) AddNode(content string, parents []string) string {
	id := fmt.Sprintf("thought_%d", g.nextID.Add(1))
	node := &ThoughtNode{
		ID:       id,
		Content:  content,
		Parents:  parents,
		Children: nil,
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	g.Nodes[id] = node
	// Update parent nodes to include this as a child
	for _, pid := range parents {
		if parent, ok := g.Nodes[pid]; ok {
			parent.Children = append(parent.Children, id)
		}
	}
	return id
}

// GetNode retrieves a node by ID.
func (g *ThoughtGraph) GetNode(id string) (*ThoughtNode, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	n, ok := g.Nodes[id]
	return n, ok
}

// LeafNodes returns all nodes with no children.
func (g *ThoughtGraph) LeafNodes() []*ThoughtNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var leaves []*ThoughtNode
	for _, n := range g.Nodes {
		if len(n.Children) == 0 {
			leaves = append(leaves, n)
		}
	}
	return leaves
}

// Controller decides the next operation to apply to the thought graph.
// It drives the graph construction process by examining the current state
// and choosing the appropriate transformation.
type Controller interface {
	// NextOperation examines the graph and returns the next operation, or nil
	// to signal that the graph is complete.
	NextOperation(ctx context.Context, graph *ThoughtGraph) (*Operation, error)
}

// DefaultController is a simple controller that generates thoughts, optionally
// merges leaves, and then aggregates. It runs a fixed sequence of operations.
type DefaultController struct {
	generateCount int
	mergeEnabled  bool
	iteration     int
	maxIterations int
}

// DefaultControllerOption configures a DefaultController.
type DefaultControllerOption func(*DefaultController)

// WithGenerateCount sets how many thoughts to generate per generate operation.
func WithGenerateCount(n int) DefaultControllerOption {
	return func(c *DefaultController) {
		if n > 0 {
			c.generateCount = n
		}
	}
}

// WithMergeEnabled enables merging of leaf nodes.
func WithMergeEnabled(enabled bool) DefaultControllerOption {
	return func(c *DefaultController) {
		c.mergeEnabled = enabled
	}
}

// WithControllerMaxIterations sets the maximum number of controller iterations.
func WithControllerMaxIterations(n int) DefaultControllerOption {
	return func(c *DefaultController) {
		if n > 0 {
			c.maxIterations = n
		}
	}
}

// NewDefaultController creates a DefaultController with the given options.
func NewDefaultController(opts ...DefaultControllerOption) *DefaultController {
	c := &DefaultController{
		generateCount: 3,
		mergeEnabled:  true,
		maxIterations: 3,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NextOperation implements Controller by following a generate → merge → aggregate sequence.
func (c *DefaultController) NextOperation(_ context.Context, graph *ThoughtGraph) (*Operation, error) {
	c.iteration++

	leaves := graph.LeafNodes()

	// Phase 1: Generate from root or existing leaves
	if c.iteration <= c.maxIterations {
		var nodeIDs []string
		if len(leaves) == 0 {
			// No nodes yet — generate from scratch (empty nodeIDs means generate from input)
			nodeIDs = nil
		} else {
			// Generate from current leaves
			for _, leaf := range leaves {
				nodeIDs = append(nodeIDs, leaf.ID)
			}
		}
		return &Operation{
			Type:    OpGenerate,
			NodeIDs: nodeIDs,
			Args:    map[string]any{"count": c.generateCount},
		}, nil
	}

	// Phase 2: Merge if enabled and there are multiple leaves
	if c.mergeEnabled && len(leaves) > 1 && c.iteration == c.maxIterations+1 {
		var ids []string
		for _, leaf := range leaves {
			ids = append(ids, leaf.ID)
		}
		return &Operation{
			Type:    OpMerge,
			NodeIDs: ids,
		}, nil
	}

	// Phase 3: Aggregate to produce final result
	if len(leaves) > 0 {
		var ids []string
		for _, leaf := range leaves {
			ids = append(ids, leaf.ID)
		}
		return &Operation{
			Type:    OpAggregate,
			NodeIDs: ids,
		}, nil
	}

	// Done
	return nil, nil
}

// GoTPlanner implements the Graph of Thought reasoning strategy. Unlike the
// linear chain or tree structures, GoT allows arbitrary graph topologies with
// merge, split, loop, and aggregate operations, enabling more complex reasoning
// patterns like synthesis and refinement.
//
// Reference: "Graph of Thoughts: Solving Elaborate Problems with Large Language Models"
// (Besta et al., 2023)
type GoTPlanner struct {
	llm           llm.ChatModel
	controller    Controller
	maxOperations int
}

// GoTOption configures a GoTPlanner.
type GoTOption func(*GoTPlanner)

// WithController sets a custom controller for the Graph of Thought planner.
func WithController(ctrl Controller) GoTOption {
	return func(p *GoTPlanner) {
		p.controller = ctrl
	}
}

// WithMaxOperations sets the maximum number of operations the planner will execute.
func WithMaxOperations(n int) GoTOption {
	return func(p *GoTPlanner) {
		if n > 0 {
			p.maxOperations = n
		}
	}
}

// NewGoTPlanner creates a new Graph of Thought planner.
func NewGoTPlanner(model llm.ChatModel, opts ...GoTOption) *GoTPlanner {
	p := &GoTPlanner{
		llm:           model,
		controller:    NewDefaultController(),
		maxOperations: 10,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Plan builds a thought graph by iteratively applying operations from the
// controller, then synthesizes a final answer from the graph.
func (p *GoTPlanner) Plan(ctx context.Context, state PlannerState) ([]Action, error) {
	graph := NewThoughtGraph()

	// Seed the graph with the input
	graph.AddNode(state.Input, nil)

	for i := 0; i < p.maxOperations; i++ {
		op, err := p.controller.NextOperation(ctx, graph)
		if err != nil {
			return nil, fmt.Errorf("graph-of-thought controller: %w", err)
		}
		if op == nil {
			break // controller signals completion
		}

		if err := p.executeOperation(ctx, state.Input, graph, op); err != nil {
			return nil, fmt.Errorf("graph-of-thought op %s: %w", op.Type, err)
		}

		// If the last operation was aggregate, we're done
		if op.Type == OpAggregate {
			break
		}
	}

	return p.synthesizeFromGraph(ctx, state, graph)
}

// Replan re-runs graph construction with observations from previous actions.
func (p *GoTPlanner) Replan(ctx context.Context, state PlannerState) ([]Action, error) {
	return p.Plan(ctx, state)
}

// executeOperation applies a single operation to the thought graph.
func (p *GoTPlanner) executeOperation(ctx context.Context, input string, graph *ThoughtGraph, op *Operation) error {
	switch op.Type {
	case OpGenerate:
		return p.opGenerate(ctx, input, graph, op)
	case OpMerge:
		return p.opMerge(ctx, input, graph, op)
	case OpSplit:
		return p.opSplit(ctx, input, graph, op)
	case OpLoop:
		return p.opLoop(ctx, input, graph, op)
	case OpAggregate:
		return p.opAggregate(ctx, input, graph, op)
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
}

// opGenerate creates new thought nodes from source nodes.
func (p *GoTPlanner) opGenerate(ctx context.Context, input string, graph *ThoughtGraph, op *Operation) error {
	count := 3
	if c, ok := op.Args["count"].(int); ok && c > 0 {
		count = c
	}

	// Gather source context
	var sourceContext string
	for _, id := range op.NodeIDs {
		if node, ok := graph.GetNode(id); ok {
			sourceContext += node.Content + "\n"
		}
	}

	prompt := fmt.Sprintf(
		"Given the following problem and existing thoughts, generate %d new distinct thoughts "+
			"that advance the reasoning.\n\n"+
			"Problem: %s\n\nExisting thoughts:\n%s\n"+
			"Generate %d new thoughts, numbered 1 through %d:",
		count, input, sourceContext, count, count,
	)

	resp, err := p.llm.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("Generate distinct reasoning thoughts. Number each thought."),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return err
	}

	thoughts := parseNumberedList(resp.Text(), count)
	for _, t := range thoughts {
		graph.AddNode(t, op.NodeIDs)
	}
	return nil
}

// opMerge combines multiple thought nodes into a synthesis.
func (p *GoTPlanner) opMerge(ctx context.Context, input string, graph *ThoughtGraph, op *Operation) error {
	var contents []string
	for _, id := range op.NodeIDs {
		if node, ok := graph.GetNode(id); ok {
			contents = append(contents, node.Content)
		}
	}

	prompt := fmt.Sprintf(
		"Synthesize the following thoughts about the problem into a single coherent insight.\n\n"+
			"Problem: %s\n\nThoughts to merge:\n", input)
	for i, c := range contents {
		prompt += fmt.Sprintf("%d. %s\n", i+1, c)
	}
	prompt += "\nProvide a single synthesized thought that combines the best insights."

	resp, err := p.llm.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("Synthesize multiple thoughts into one coherent insight."),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return err
	}

	graph.AddNode(resp.Text(), op.NodeIDs)
	return nil
}

// opSplit divides a thought node into sub-thoughts.
func (p *GoTPlanner) opSplit(ctx context.Context, input string, graph *ThoughtGraph, op *Operation) error {
	if len(op.NodeIDs) == 0 {
		return nil
	}

	node, ok := graph.GetNode(op.NodeIDs[0])
	if !ok {
		return nil
	}

	prompt := fmt.Sprintf(
		"The following thought about the problem needs to be broken down into "+
			"more specific sub-thoughts.\n\n"+
			"Problem: %s\n\nThought: %s\n\n"+
			"Break this into 2-3 specific sub-thoughts, numbered:",
		input, node.Content,
	)

	resp, err := p.llm.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("Break a thought into specific sub-thoughts."),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return err
	}

	subThoughts := parseNumberedList(resp.Text(), 3)
	for _, t := range subThoughts {
		graph.AddNode(t, []string{node.ID})
	}
	return nil
}

// opLoop refines a thought node by re-evaluating it.
func (p *GoTPlanner) opLoop(ctx context.Context, input string, graph *ThoughtGraph, op *Operation) error {
	if len(op.NodeIDs) == 0 {
		return nil
	}

	node, ok := graph.GetNode(op.NodeIDs[0])
	if !ok {
		return nil
	}

	prompt := fmt.Sprintf(
		"Refine and improve the following thought about the problem.\n\n"+
			"Problem: %s\n\nOriginal thought: %s\n\n"+
			"Provide an improved version of this thought with better reasoning:",
		input, node.Content,
	)

	resp, err := p.llm.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("Refine and improve a reasoning thought."),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return err
	}

	graph.AddNode(resp.Text(), []string{node.ID})
	return nil
}

// opAggregate collects all specified nodes and produces a final synthesis.
func (p *GoTPlanner) opAggregate(ctx context.Context, input string, graph *ThoughtGraph, op *Operation) error {
	var contents []string
	for _, id := range op.NodeIDs {
		if node, ok := graph.GetNode(id); ok {
			contents = append(contents, node.Content)
		}
	}

	prompt := fmt.Sprintf(
		"Aggregate the following thoughts into a final comprehensive answer.\n\n"+
			"Problem: %s\n\nThoughts:\n", input)
	for i, c := range contents {
		prompt += fmt.Sprintf("%d. %s\n", i+1, c)
	}
	prompt += "\nProvide a comprehensive final answer."

	resp, err := p.llm.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("Aggregate thoughts into a comprehensive answer."),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return err
	}

	graph.AddNode(resp.Text(), op.NodeIDs)
	return nil
}

// synthesizeFromGraph produces the final actions from the thought graph.
func (p *GoTPlanner) synthesizeFromGraph(ctx context.Context, state PlannerState, graph *ThoughtGraph) ([]Action, error) {
	// Use the latest leaf nodes as the reasoning context
	leaves := graph.LeafNodes()

	var reasoning strings.Builder
	reasoning.WriteString("Reasoning from thought graph:\n\n")
	for i, leaf := range leaves {
		fmt.Fprintf(&reasoning, "Insight %d: %s\n", i+1, leaf.Content)
	}

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
		return nil, fmt.Errorf("graph-of-thought synthesize: %w", err)
	}

	return parseAIResponse(resp), nil
}

// parseNumberedList extracts items from a numbered list response, up to maxItems.
func parseNumberedList(text string, maxItems int) []string {
	lines := strings.Split(text, "\n")
	var items []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		cleaned := strings.TrimLeft(line, "0123456789. ")
		if cleaned != "" {
			items = append(items, cleaned)
		}
		if len(items) >= maxItems {
			break
		}
	}
	return items
}
