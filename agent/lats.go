package agent

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	RegisterPlanner("lats", func(cfg PlannerConfig) (Planner, error) {
		if cfg.LLM == nil {
			return nil, fmt.Errorf("lats planner requires an LLM")
		}
		var opts []LATSOption
		if ew, ok := cfg.Extra["expansion_width"].(int); ok {
			opts = append(opts, WithExpansionWidth(ew))
		}
		if md, ok := cfg.Extra["max_depth"].(int); ok {
			opts = append(opts, WithLATSMaxDepth(md))
		}
		if ec, ok := cfg.Extra["exploration_constant"].(float64); ok {
			opts = append(opts, WithExplorationConstant(ec))
		}
		return NewLATSPlanner(cfg.LLM, opts...), nil
	})
}

// MCTSNode represents a node in the Monte Carlo Tree Search used by LATS.
type MCTSNode struct {
	// State is the reasoning state at this node.
	State string
	// Children are the child nodes expanded from this node.
	Children []*MCTSNode
	// Parent is the parent node (nil for root).
	Parent *MCTSNode
	// Visits is the number of times this node has been visited.
	Visits int
	// Value is the accumulated value from simulations through this node.
	Value float64
	// Depth is the depth of this node in the tree.
	Depth int
	// Reflection holds verbal feedback from failed evaluations.
	Reflection string
}

// UCTScore calculates the Upper Confidence bound for Trees score for this node.
// It balances exploitation (high value) with exploration (low visits).
func (n *MCTSNode) UCTScore(explorationConstant float64) float64 {
	if n.Visits == 0 {
		return math.Inf(1) // prioritize unvisited nodes
	}
	if n.Parent == nil {
		return 0
	}
	exploitation := n.Value / float64(n.Visits)
	exploration := explorationConstant * math.Sqrt(math.Log(float64(n.Parent.Visits))/float64(n.Visits))
	return exploitation + exploration
}

// LATSPlanner implements Language Agent Tree Search, combining Monte Carlo
// Tree Search (MCTS) with LLM-based reasoning. It uses selection (UCT),
// expansion (generating N candidates), evaluation (scoring), and
// backpropagation to systematically explore the reasoning space. On failure,
// it generates verbal reflections to guide future search iterations.
//
// Reference: "Language Agent Tree Search Unifies Reasoning, Acting, and Planning
// in Language Models" (Zhou et al., 2023)
type LATSPlanner struct {
	llm                 llm.ChatModel
	expansionWidth      int
	maxDepth            int
	explorationConstant float64
	reflections         []string
}

// LATSOption configures a LATSPlanner.
type LATSOption func(*LATSPlanner)

// WithExpansionWidth sets the number of candidate nodes generated per expansion.
func WithExpansionWidth(n int) LATSOption {
	return func(p *LATSPlanner) {
		if n > 0 {
			p.expansionWidth = n
		}
	}
}

// WithLATSMaxDepth sets the maximum depth of the MCTS tree.
func WithLATSMaxDepth(n int) LATSOption {
	return func(p *LATSPlanner) {
		if n > 0 {
			p.maxDepth = n
		}
	}
}

// WithExplorationConstant sets the exploration constant for the UCT formula.
// Higher values encourage more exploration over exploitation.
func WithExplorationConstant(c float64) LATSOption {
	return func(p *LATSPlanner) {
		if c > 0 {
			p.explorationConstant = c
		}
	}
}

// NewLATSPlanner creates a new LATS planner.
func NewLATSPlanner(model llm.ChatModel, opts ...LATSOption) *LATSPlanner {
	p := &LATSPlanner{
		llm:                 model,
		expansionWidth:      5,
		maxDepth:            10,
		explorationConstant: 1.41, // sqrt(2), standard UCT constant
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Plan performs MCTS-guided search over the reasoning space.
func (p *LATSPlanner) Plan(ctx context.Context, state PlannerState) ([]Action, error) {
	root := &MCTSNode{
		State: state.Input,
		Depth: 0,
	}

	// Run MCTS iterations
	iterations := p.expansionWidth * p.maxDepth
	for i := 0; i < iterations; i++ {
		// 1. Selection — use UCT to find the most promising leaf
		leaf := p.selectNode(root)

		// 2. Expansion — generate child nodes
		if leaf.Depth < p.maxDepth {
			if err := p.expandNode(ctx, state.Input, leaf); err != nil {
				continue // skip failed expansions
			}

			// Select a newly created child
			if len(leaf.Children) > 0 {
				leaf = leaf.Children[0]
			}
		}

		// 3. Evaluation — score the leaf node
		score, err := p.evaluateNode(ctx, state.Input, leaf)
		if err != nil {
			continue
		}

		// 4. Backpropagation — update values up the tree
		p.backpropagate(leaf, score)

		// If we found a high-confidence path, use it
		if score >= 0.9 {
			path := p.extractPath(leaf)
			return p.synthesize(ctx, state, path)
		}

		// If score is low, generate a reflection
		if score < 0.3 && leaf.Reflection == "" {
			reflection, err := p.reflect(ctx, state.Input, leaf.State, score)
			if err == nil {
				leaf.Reflection = reflection
				p.reflections = append(p.reflections, reflection)
			}
		}
	}

	// Use the best path found
	bestLeaf := p.bestLeaf(root)
	path := p.extractPath(bestLeaf)
	return p.synthesize(ctx, state, path)
}

// Replan re-runs the MCTS search with accumulated reflections.
func (p *LATSPlanner) Replan(ctx context.Context, state PlannerState) ([]Action, error) {
	return p.Plan(ctx, state)
}

// selectNode traverses the tree using UCT to find the most promising leaf node.
func (p *LATSPlanner) selectNode(node *MCTSNode) *MCTSNode {
	for len(node.Children) > 0 {
		var bestChild *MCTSNode
		bestScore := math.Inf(-1)

		for _, child := range node.Children {
			score := child.UCTScore(p.explorationConstant)
			if score > bestScore {
				bestScore = score
				bestChild = child
			}
		}

		if bestChild == nil {
			break
		}
		node = bestChild
	}
	return node
}

// expandNode generates candidate child nodes for the given node.
func (p *LATSPlanner) expandNode(ctx context.Context, input string, node *MCTSNode) error {
	var reflectionContext string
	if len(p.reflections) > 0 {
		reflectionContext = "\n\nPrevious reflections to consider:\n" + strings.Join(p.reflections, "\n")
	}

	// Build the path context
	path := p.extractPath(node)
	var pathContext string
	if len(path) > 1 { // skip root
		pathContext = "\n\nReasoning so far:\n"
		for i, step := range path[1:] {
			pathContext += fmt.Sprintf("Step %d: %s\n", i+1, step)
		}
	}

	prompt := fmt.Sprintf(
		"Generate %d distinct next reasoning steps for the following problem.%s%s\n\n"+
			"Problem: %s\n\nCurrent state: %s\n\n"+
			"Generate %d different possible next steps. Number them 1 through %d.",
		p.expansionWidth, pathContext, reflectionContext,
		input, node.State, p.expansionWidth, p.expansionWidth,
	)

	resp, err := p.llm.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("You are an expert problem solver. Generate diverse reasoning steps."),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return err
	}

	thoughts := parseNumberedList(resp.Text(), p.expansionWidth)
	for _, t := range thoughts {
		child := &MCTSNode{
			State:  t,
			Parent: node,
			Depth:  node.Depth + 1,
		}
		node.Children = append(node.Children, child)
	}

	return nil
}

// evaluateNode scores a node's reasoning state.
func (p *LATSPlanner) evaluateNode(ctx context.Context, input string, node *MCTSNode) (float64, error) {
	path := p.extractPath(node)
	var pathText string
	for i, step := range path {
		pathText += fmt.Sprintf("Step %d: %s\n", i+1, step)
	}

	prompt := fmt.Sprintf(
		"Evaluate the quality of the following reasoning path for solving the problem.\n\n"+
			"Problem: %s\n\nReasoning path:\n%s\n"+
			"Rate the quality on a scale of 0.0 to 1.0, where:\n"+
			"- 0.0 = completely wrong or irrelevant\n"+
			"- 0.5 = partially correct but incomplete\n"+
			"- 1.0 = correct and complete solution\n\n"+
			"Reply with ONLY a number between 0.0 and 1.0.",
		input, pathText,
	)

	resp, err := p.llm.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("You are a reasoning evaluator. Output only a decimal number between 0.0 and 1.0."),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return 0, err
	}

	text := strings.TrimSpace(resp.Text())
	var score float64
	if _, err := fmt.Sscanf(text, "%f", &score); err != nil {
		return 0.5, nil // default to middle score if parse fails
	}

	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score, nil
}

// backpropagate updates the value and visit count from a leaf up to the root.
func (p *LATSPlanner) backpropagate(node *MCTSNode, value float64) {
	for n := node; n != nil; n = n.Parent {
		n.Visits++
		n.Value += value
	}
}

// reflect generates a verbal reflection on a low-scoring reasoning path.
func (p *LATSPlanner) reflect(ctx context.Context, input, state string, score float64) (string, error) {
	prompt := fmt.Sprintf(
		"The following reasoning step scored %.2f/1.0 for the problem.\n\n"+
			"Problem: %s\n\nReasoning: %s\n\n"+
			"Provide a brief reflection on what went wrong and how to improve.",
		score, input, state,
	)

	resp, err := p.llm.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("Provide concise, actionable reflection on failed reasoning."),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return "", err
	}
	return resp.Text(), nil
}

// extractPath builds the reasoning path from root to the given node.
func (p *LATSPlanner) extractPath(node *MCTSNode) []string {
	var path []string
	for n := node; n != nil; n = n.Parent {
		path = append(path, n.State)
	}
	// Reverse to get root-to-leaf order
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// bestLeaf finds the leaf node with the highest average value.
func (p *LATSPlanner) bestLeaf(root *MCTSNode) *MCTSNode {
	best := root
	bestAvg := -1.0

	var dfs func(node *MCTSNode)
	dfs = func(node *MCTSNode) {
		if len(node.Children) == 0 && node.Visits > 0 {
			avg := node.Value / float64(node.Visits)
			if avg > bestAvg {
				bestAvg = avg
				best = node
			}
		}
		for _, child := range node.Children {
			dfs(child)
		}
	}
	dfs(root)

	return best
}

// synthesize generates the final response using the best reasoning path discovered.
func (p *LATSPlanner) synthesize(ctx context.Context, state PlannerState, path []string) ([]Action, error) {
	messages := buildMessagesFromState(state)

	var pathText strings.Builder
	for i, step := range path {
		if i == 0 {
			continue // skip root (which is just the input)
		}
		fmt.Fprintf(&pathText, "Step %d: %s\n", i, step)
	}

	var reflectionText string
	if len(p.reflections) > 0 {
		reflectionText = "\n\nLessons learned from exploration:\n" + strings.Join(p.reflections, "\n")
	}

	structureMsg := schema.NewSystemMessage(
		"Use the following reasoning path to formulate your response:\n\n" +
			pathText.String() + reflectionText,
	)

	msgs := make([]schema.Message, 0, len(messages)+1)
	msgs = append(msgs, structureMsg)
	msgs = append(msgs, messages...)

	model := p.llm
	if len(state.Tools) > 0 {
		model = model.BindTools(toolDefinitions(state.Tools))
	}

	resp, err := model.Generate(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("lats synthesize: %w", err)
	}

	return parseAIResponse(resp), nil
}

// Reflections returns the accumulated reflections (for inspection/testing).
func (p *LATSPlanner) Reflections() []string {
	return p.reflections
}
