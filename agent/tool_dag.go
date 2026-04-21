package agent

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/lookatitude/beluga-ai/v2/tool"
)

// ToolDAGExecutor runs a slice of tool calls with dependency-aware parallelism.
// Independent calls execute concurrently (bounded by MaxConcurrency); calls whose
// arguments reference the ID of an earlier call are deferred until that call's
// result is available.
type ToolDAGExecutor struct {
	maxConcurrency      int
	dependencyDetection bool
}

// ToolDAGOption is a functional option for ToolDAGExecutor.
type ToolDAGOption func(*ToolDAGExecutor)

// WithMaxConcurrency sets the maximum number of tool calls that may execute at
// the same time. A value of 0 or less is ignored; the current value is kept.
func WithMaxConcurrency(n int) ToolDAGOption {
	return func(e *ToolDAGExecutor) {
		if n > 0 {
			e.maxConcurrency = n
		}
	}
}

// WithDependencyDetection controls whether the executor scans each call's
// arguments for references to other call IDs and orders them accordingly.
// When false (the default) every call is treated as independent and all run
// in parallel up to MaxConcurrency.
func WithDependencyDetection(enabled bool) ToolDAGOption {
	return func(e *ToolDAGExecutor) {
		e.dependencyDetection = enabled
	}
}

// NewToolDAGExecutor creates a ToolDAGExecutor with the given options. The
// default MaxConcurrency is 8 and dependency detection is disabled.
func NewToolDAGExecutor(opts ...ToolDAGOption) *ToolDAGExecutor {
	e := &ToolDAGExecutor{
		maxConcurrency:      8,
		dependencyDetection: false,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Execute runs all calls using the provided registry and returns one *tool.Result
// per call in the same order as the input slice. When dependency detection is
// disabled all calls run in parallel (bounded by MaxConcurrency). When enabled,
// the executor analyses each call's JSON arguments for string values that match
// another call's ID and orders accordingly.
//
// Context cancellation stops pending calls; already-running calls run to
// completion (or respect the context themselves).
func (e *ToolDAGExecutor) Execute(ctx context.Context, calls []schema.ToolCall, registry *tool.Registry) []*tool.Result {
	if len(calls) == 0 {
		return nil
	}

	results := make([]*tool.Result, len(calls))

	if e.dependencyDetection {
		e.executeWithDAG(ctx, calls, registry, results)
	} else {
		e.executeParallel(ctx, calls, registry, results)
	}

	return results
}

// executeParallel runs every call concurrently, bounded by maxConcurrency.
func (e *ToolDAGExecutor) executeParallel(ctx context.Context, calls []schema.ToolCall, registry *tool.Registry, results []*tool.Result) {
	sem := make(chan struct{}, e.maxConcurrency)
	var wg sync.WaitGroup

	for i, call := range calls {
		if ctx.Err() != nil {
			results[i] = cancelledResult()
			continue
		}

		wg.Add(1)
		go func(idx int, tc schema.ToolCall) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			results[idx] = runOneTool(ctx, tc, registry)
		}(i, call)
	}

	wg.Wait()
}

// dagNode represents one tool call in the dependency graph.
type dagNode struct {
	call  schema.ToolCall
	index int   // position in the original calls slice
	deps  []int // indices of nodes that must complete before this one
}

// dagExecState groups the mutable scheduling state used during DAG execution.
// Passing it as a single value reduces parameter counts on inner functions.
type dagExecState struct {
	mu         *sync.Mutex
	ready      *[]int
	inDegree   []int
	dependents [][]int
}

// executeWithDAG builds a dependency graph from call argument references, then
// executes nodes level by level, running each level's members in parallel.
func (e *ToolDAGExecutor) executeWithDAG(ctx context.Context, calls []schema.ToolCall, registry *tool.Registry, results []*tool.Result) {
	nodes := e.buildDAG(calls)

	// inDegree counts unsatisfied dependencies for each node.
	inDegree := make([]int, len(nodes))
	// dependents maps node index → indices that depend on it.
	dependents := make([][]int, len(nodes))
	for i := range dependents {
		dependents[i] = []int{}
	}

	for i, n := range nodes {
		inDegree[i] = len(n.deps)
		for _, dep := range n.deps {
			dependents[dep] = append(dependents[dep], i)
		}
	}

	// Collect nodes with no dependencies as the first ready set.
	ready := make([]int, 0, len(nodes))
	for i, d := range inDegree {
		if d == 0 {
			ready = append(ready, i)
		}
	}

	state := dagExecState{
		mu:         &sync.Mutex{},
		ready:      &ready,
		inDegree:   inDegree,
		dependents: dependents,
	}

	completed := 0
	total := len(nodes)

	for completed < total {
		if ctx.Err() != nil {
			cancelPendingNodes(nodes, results)
			return
		}

		state.mu.Lock()
		batch := make([]int, len(*state.ready))
		copy(batch, *state.ready)
		*state.ready = (*state.ready)[:0]
		state.mu.Unlock()

		if len(batch) == 0 {
			// Cycle or other issue — break to avoid deadlock.
			break
		}

		e.runBatch(ctx, batch, nodes, registry, results, state)
		completed += len(batch)
	}

	// Safety net: fill any slots that were never scheduled (e.g. due to a cycle).
	for _, n := range nodes {
		if results[n.index] == nil {
			results[n.index] = runOneTool(ctx, n.call, registry)
		}
	}
}

// runBatch executes the nodes in batch in parallel (bounded by maxConcurrency),
// then updates the ready queue with any nodes that become unblocked.
func (e *ToolDAGExecutor) runBatch(
	ctx context.Context,
	batch []int,
	nodes []dagNode,
	registry *tool.Registry,
	results []*tool.Result,
	st dagExecState,
) {
	sem := make(chan struct{}, e.maxConcurrency)
	var wg sync.WaitGroup

	for _, nodeIdx := range batch {
		n := nodes[nodeIdx]

		if ctx.Err() != nil {
			results[n.index] = cancelledResult()
			unblockDependents(nodeIdx, st)
			continue
		}

		wg.Add(1)
		go func(ni int, nd dagNode) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			results[nd.index] = runOneTool(ctx, nd.call, registry)
			unblockDependents(ni, st)
		}(nodeIdx, n)
	}

	wg.Wait()
}

// unblockDependents decrements the in-degree of each node that depends on
// nodeIdx and appends any newly ready nodes to the ready queue.
func unblockDependents(nodeIdx int, st dagExecState) {
	st.mu.Lock()
	for _, dep := range st.dependents[nodeIdx] {
		st.inDegree[dep]--
		if st.inDegree[dep] == 0 {
			*st.ready = append(*st.ready, dep)
		}
	}
	st.mu.Unlock()
}

// cancelPendingNodes fills every nil result slot with a cancelled result.
func cancelPendingNodes(nodes []dagNode, results []*tool.Result) {
	for _, n := range nodes {
		if results[n.index] == nil {
			// Use n.index (position in the original calls slice), not the
			// loop variable position in the nodes slice, which may differ
			// when the DAG reorders nodes.
			results[n.index] = cancelledResult()
		}
	}
}

// buildDAG converts a flat slice of tool calls into dagNodes, detecting
// dependencies by scanning each call's JSON arguments for string values that
// match another call's ID.
func (e *ToolDAGExecutor) buildDAG(calls []schema.ToolCall) []dagNode {
	idToIdx := make(map[string]int, len(calls))
	for i, c := range calls {
		if c.ID != "" {
			idToIdx[c.ID] = i
		}
	}

	nodes := make([]dagNode, len(calls))
	for i, c := range calls {
		deps := detectDependencies(c.Arguments, idToIdx, i)
		nodes[i] = dagNode{call: c, index: i, deps: deps}
	}
	return nodes
}

// detectDependencies returns the indices (from idToIdx) of call IDs referenced
// as string values anywhere inside rawJSON, excluding selfIdx.
func detectDependencies(rawJSON string, idToIdx map[string]int, selfIdx int) []int {
	if rawJSON == "" {
		return nil
	}

	var parsed any
	if err := json.Unmarshal([]byte(rawJSON), &parsed); err != nil {
		return nil
	}

	seen := map[int]bool{}
	var walk func(v any)
	walk = func(v any) {
		switch val := v.(type) {
		case string:
			if idx, ok := idToIdx[val]; ok && idx != selfIdx {
				seen[idx] = true
			}
		case map[string]any:
			for _, child := range val {
				walk(child)
			}
		case []any:
			for _, child := range val {
				walk(child)
			}
		}
	}
	walk(parsed)

	if len(seen) == 0 {
		return nil
	}
	result := make([]int, 0, len(seen))
	for idx := range seen {
		result = append(result, idx)
	}
	return result
}

// runOneTool looks up the named tool in registry, parses JSON arguments, and
// executes it. Any error is encoded as an IsError result so the caller always
// receives a non-nil value.
func runOneTool(ctx context.Context, call schema.ToolCall, registry *tool.Registry) *tool.Result {
	if ctx.Err() != nil {
		return cancelledResult()
	}

	t, err := registry.Get(call.Name)
	if err != nil {
		return tool.ErrorResult(core.Errorf(core.ErrNotFound, "tool %q not found: %w", call.Name, err))
	}

	var args map[string]any
	if strings.TrimSpace(call.Arguments) != "" {
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return tool.ErrorResult(core.Errorf(core.ErrInvalidInput, "invalid arguments for tool %q: %w", call.Name, err))
		}
	}

	res, err := t.Execute(ctx, args)
	if err != nil {
		return tool.ErrorResult(core.Errorf(core.ErrProviderDown, "tool %q execution failed: %w", call.Name, err))
	}
	return res
}

// cancelledResult returns an IsError result representing context cancellation.
func cancelledResult() *tool.Result {
	return tool.ErrorResult(context.Canceled)
}
