package agent

import (
	"container/heap"
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	RegisterPlanner("tree-of-thought", func(cfg PlannerConfig) (Planner, error) {
		if cfg.LLM == nil {
			return nil, fmt.Errorf("tree-of-thought planner requires an LLM")
		}
		var opts []ToTOption
		if bf, ok := cfg.Extra["branch_factor"].(int); ok {
			opts = append(opts, WithBranchFactor(bf))
		}
		if md, ok := cfg.Extra["max_depth"].(int); ok {
			opts = append(opts, WithMaxDepth(md))
		}
		if s, ok := cfg.Extra["strategy"].(SearchStrategy); ok {
			opts = append(opts, WithSearchStrategy(s))
		}
		return NewToTPlanner(cfg.LLM, opts...), nil
	})
}

// SearchStrategy determines the tree traversal method for Tree of Thought.
type SearchStrategy string

const (
	// StrategyBFS uses breadth-first search, exploring the most promising nodes
	// at each level before going deeper.
	StrategyBFS SearchStrategy = "bfs"
	// StrategyDFS uses depth-first search, exploring a single path to completion
	// before backtracking to explore alternatives.
	StrategyDFS SearchStrategy = "dfs"
)

// ToTPlanner implements the Tree of Thought reasoning strategy. It explores
// multiple reasoning paths by generating candidate thoughts at each step,
// evaluating their promise, and expanding the most promising branches.
//
// Reference: "Tree of Thoughts: Deliberate Problem Solving with Large Language Models"
// (Yao et al., 2023)
type ToTPlanner struct {
	llm          llm.ChatModel
	branchFactor int
	maxDepth     int
	strategy     SearchStrategy
}

// ToTOption configures a ToTPlanner.
type ToTOption func(*ToTPlanner)

// WithBranchFactor sets the number of candidate thoughts generated per node.
func WithBranchFactor(n int) ToTOption {
	return func(p *ToTPlanner) {
		if n > 0 {
			p.branchFactor = n
		}
	}
}

// WithMaxDepth sets the maximum depth of the thought tree.
func WithMaxDepth(n int) ToTOption {
	return func(p *ToTPlanner) {
		if n > 0 {
			p.maxDepth = n
		}
	}
}

// WithSearchStrategy sets the search strategy (BFS or DFS).
func WithSearchStrategy(s SearchStrategy) ToTOption {
	return func(p *ToTPlanner) {
		p.strategy = s
	}
}

// NewToTPlanner creates a new Tree of Thought planner.
func NewToTPlanner(model llm.ChatModel, opts ...ToTOption) *ToTPlanner {
	p := &ToTPlanner{
		llm:          model,
		branchFactor: 3,
		maxDepth:     5,
		strategy:     StrategyBFS,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// thoughtNode represents a single node in the thought tree.
type thoughtNode struct {
	thought  string
	score    float64
	depth    int
	path     []string // chain of thoughts from root to this node
	priority int      // for heap ordering
	index    int      // heap index
}

// thoughtHeap implements a max-heap of thought nodes ordered by score.
type thoughtHeap []*thoughtNode

func (h thoughtHeap) Len() int            { return len(h) }
func (h thoughtHeap) Less(i, j int) bool   { return h[i].score > h[j].score } // max-heap
func (h thoughtHeap) Swap(i, j int)        { h[i], h[j] = h[j], h[i]; h[i].index = i; h[j].index = j }
func (h *thoughtHeap) Push(x any)          { n := x.(*thoughtNode); n.index = len(*h); *h = append(*h, n) }
func (h *thoughtHeap) Pop() any            { old := *h; n := old[len(old)-1]; old[len(old)-1] = nil; n.index = -1; *h = old[:len(old)-1]; return n }

// Plan explores the thought tree to find the best reasoning path, then
// generates a final response using that path.
func (p *ToTPlanner) Plan(ctx context.Context, state PlannerState) ([]Action, error) {
	var bestPath []string
	var err error

	switch p.strategy {
	case StrategyDFS:
		bestPath, err = p.searchDFS(ctx, state)
	default: // BFS
		bestPath, err = p.searchBFS(ctx, state)
	}
	if err != nil {
		return nil, fmt.Errorf("tree-of-thought search: %w", err)
	}

	return p.synthesize(ctx, state, bestPath)
}

// Replan re-runs the search with observations from previous actions.
func (p *ToTPlanner) Replan(ctx context.Context, state PlannerState) ([]Action, error) {
	return p.Plan(ctx, state)
}

// searchBFS performs breadth-first search over the thought tree using a priority queue.
func (p *ToTPlanner) searchBFS(ctx context.Context, state PlannerState) ([]string, error) {
	h := &thoughtHeap{}
	heap.Init(h)

	// Generate and enqueue initial thoughts
	thoughts, err := p.generateThoughts(ctx, state.Input, nil)
	if err != nil {
		return nil, err
	}
	p.enqueueThoughts(ctx, h, thoughts, state.Input, nil, 1)

	var bestPath []string
	bestScore := -1.0

	for h.Len() > 0 {
		node := heap.Pop(h).(*thoughtNode)

		if node.score > bestScore {
			bestScore = node.score
			bestPath = node.path
		}

		if node.depth >= p.maxDepth {
			continue
		}

		children, err := p.generateThoughts(ctx, state.Input, node.path)
		if err != nil {
			continue
		}
		p.enqueueThoughts(ctx, h, children, state.Input, node.path, node.depth+1)
	}

	if bestPath == nil {
		return []string{"Unable to find a viable reasoning path."}, nil
	}

	return bestPath, nil
}

// enqueueThoughts evaluates each thought and pushes viable ones onto the heap.
func (p *ToTPlanner) enqueueThoughts(ctx context.Context, h *thoughtHeap, thoughts []string, input string, parentPath []string, depth int) {
	for _, t := range thoughts {
		score, err := p.evaluateThought(ctx, input, t)
		if err != nil || score <= 0 {
			continue
		}

		path := make([]string, len(parentPath)+1)
		if len(parentPath) > 0 {
			copy(path, parentPath)
		}
		path[len(parentPath)] = t

		heap.Push(h, &thoughtNode{
			thought: t,
			score:   score,
			depth:   depth,
			path:    path,
		})
	}
}

// searchDFS performs depth-first search over the thought tree.
func (p *ToTPlanner) searchDFS(ctx context.Context, state PlannerState) ([]string, error) {
	result, _ := p.dfsHelper(ctx, state.Input, nil, 0, -1.0)
	if result == nil {
		return []string{"Unable to find a viable reasoning path."}, nil
	}
	return result, nil
}

// dfsHelper recursively explores paths depth-first, returning the best path and its score.
func (p *ToTPlanner) dfsHelper(ctx context.Context, input string, path []string, depth int, bestScore float64) ([]string, float64) {
	if depth >= p.maxDepth {
		return path, bestScore
	}

	thoughts, err := p.generateThoughts(ctx, input, path)
	if err != nil {
		return path, bestScore
	}

	currentBest := path
	currentBestScore := bestScore

	for _, t := range thoughts {
		score, err := p.evaluateThought(ctx, input, t)
		if err != nil || score <= 0 {
			continue
		}

		childPath := make([]string, len(path)+1)
		copy(childPath, path)
		childPath[len(path)] = t

		if score > currentBestScore {
			deeper, deeperScore := p.dfsHelper(ctx, input, childPath, depth+1, score)
			if deeperScore > currentBestScore {
				currentBest = deeper
				currentBestScore = deeperScore
			}
		}
	}

	return currentBest, currentBestScore
}

// generateThoughts asks the LLM to generate candidate thoughts given the current path.
func (p *ToTPlanner) generateThoughts(ctx context.Context, input string, path []string) ([]string, error) {
	var pathContext string
	if len(path) > 0 {
		pathContext = "\n\nCurrent reasoning path:\n"
		for i, step := range path {
			pathContext += fmt.Sprintf("Step %d: %s\n", i+1, step)
		}
	}

	prompt := fmt.Sprintf(
		"Given the following problem, generate exactly %d distinct next reasoning steps.%s\n\n"+
			"Problem: %s\n\n"+
			"Generate %d different possible next steps in the reasoning. "+
			"Each step should be a single clear thought. "+
			"Number them 1 through %d, one per line.",
		p.branchFactor, pathContext, input, p.branchFactor, p.branchFactor,
	)

	resp, err := p.llm.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("You are an expert problem solver. Generate distinct reasoning steps."),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return nil, err
	}

	// Parse numbered thoughts
	lines := strings.Split(resp.Text(), "\n")
	var thoughts []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Strip numbering prefix
		cleaned := strings.TrimLeft(line, "0123456789. ")
		if cleaned != "" {
			thoughts = append(thoughts, cleaned)
		}
	}

	// Limit to branch factor
	if len(thoughts) > p.branchFactor {
		thoughts = thoughts[:p.branchFactor]
	}

	return thoughts, nil
}

// evaluateThought asks the LLM to evaluate a thought as "sure" (1.0), "maybe" (0.5),
// or "impossible" (0.0).
func (p *ToTPlanner) evaluateThought(ctx context.Context, input string, thought string) (float64, error) {
	prompt := fmt.Sprintf(
		"Evaluate whether the following reasoning step is on the right track "+
			"for solving the problem.\n\n"+
			"Problem: %s\n\nReasoning step: %s\n\n"+
			"Evaluate this step as one of:\n"+
			"- \"sure\" — this step is clearly correct and productive\n"+
			"- \"maybe\" — this step could be useful but uncertain\n"+
			"- \"impossible\" — this step is clearly wrong or a dead end\n\n"+
			"Reply with ONLY one word: sure, maybe, or impossible.",
		input, thought,
	)

	resp, err := p.llm.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("You are a reasoning evaluator. Reply with exactly one word: sure, maybe, or impossible."),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return 0, err
	}

	text := strings.TrimSpace(strings.ToLower(resp.Text()))
	switch {
	case strings.Contains(text, "sure"):
		return 1.0, nil
	case strings.Contains(text, "maybe"):
		return 0.5, nil
	default:
		return 0.0, nil
	}
}

// synthesize generates the final response using the best reasoning path discovered.
func (p *ToTPlanner) synthesize(ctx context.Context, state PlannerState, path []string) ([]Action, error) {
	messages := buildMessagesFromState(state)

	var pathText strings.Builder
	for i, step := range path {
		fmt.Fprintf(&pathText, "Step %d: %s\n", i+1, step)
	}

	structureMsg := schema.NewSystemMessage(
		"Use the following reasoning path to formulate your response:\n\n" + pathText.String(),
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
		return nil, fmt.Errorf("tree-of-thought synthesize: %w", err)
	}

	return parseAIResponse(resp), nil
}
