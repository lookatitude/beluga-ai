package graph

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
)

// BasicGraph provides a basic implementation of the Graph interface
type BasicGraph struct {
	config     iface.GraphConfig
	nodes      map[string]core.Runnable
	edges      map[string][]string // node -> []target_nodes
	entryNodes []string
	exitNodes  []string
	tracer     trace.Tracer
	mu         sync.RWMutex
}

// NewBasicGraph creates a new BasicGraph
func NewBasicGraph(config iface.GraphConfig, tracer trace.Tracer) *BasicGraph {
	return &BasicGraph{
		config:     config,
		nodes:      make(map[string]core.Runnable),
		edges:      make(map[string][]string),
		entryNodes: config.EntryPoints,
		exitNodes:  config.ExitPoints,
		tracer:     tracer,
	}
}

func (g *BasicGraph) AddNode(name string, runnable core.Runnable) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.nodes[name]; exists {
		return iface.ErrInvalidConfig("graph.add_node", fmt.Errorf("node %s already exists", name))
	}

	g.nodes[name] = runnable
	return nil
}

func (g *BasicGraph) AddEdge(sourceNode string, targetNode string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Validate that both nodes exist
	if _, exists := g.nodes[sourceNode]; !exists {
		return iface.ErrNotFound("graph.add_edge", fmt.Sprintf("source node %s", sourceNode))
	}
	if _, exists := g.nodes[targetNode]; !exists {
		return iface.ErrNotFound("graph.add_edge", fmt.Sprintf("target node %s", targetNode))
	}

	// Add edge
	if g.edges[sourceNode] == nil {
		g.edges[sourceNode] = make([]string, 0)
	}
	g.edges[sourceNode] = append(g.edges[sourceNode], targetNode)

	return nil
}

func (g *BasicGraph) SetEntryPoint(nodeNames []string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Validate that all entry nodes exist
	for _, name := range nodeNames {
		if _, exists := g.nodes[name]; !exists {
			return iface.ErrNotFound("graph.set_entry_point", fmt.Sprintf("entry node %s", name))
		}
	}

	g.entryNodes = make([]string, len(nodeNames))
	copy(g.entryNodes, nodeNames)
	return nil
}

func (g *BasicGraph) SetFinishPoint(nodeNames []string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Validate that all exit nodes exist
	for _, name := range nodeNames {
		if _, exists := g.nodes[name]; !exists {
			return iface.ErrNotFound("graph.set_finish_point", fmt.Sprintf("exit node %s", name))
		}
	}

	g.exitNodes = make([]string, len(nodeNames))
	copy(g.exitNodes, nodeNames)
	return nil
}

func (g *BasicGraph) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	var span trace.Span
	if g.tracer != nil {
		ctx, span = g.tracer.Start(ctx, "graph.invoke",
			trace.WithAttributes(
				attribute.String("graph.name", g.config.Name),
				attribute.Int("graph.nodes", len(g.nodes)),
				attribute.Int("graph.edges", g.countEdges()),
			))
		defer span.End()
	}

	startTime := time.Now()
	var err error
	defer func() {
		duration := time.Since(startTime).Seconds()
		if g.tracer != nil && span != nil {
			if err != nil {
				span.RecordError(err)
			}
			span.SetAttributes(attribute.Float64("graph.duration", duration))
		}
	}()

	// Apply timeout if configured
	if g.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(g.config.Timeout)*time.Second)
		defer cancel()
	}

	// Execute graph based on execution mode
	if g.config.EnableParallelExecution {
		return g.executeParallel(ctx, input, options...)
	}
	return g.executeSequential(ctx, input, options...)
}

func (g *BasicGraph) executeSequential(ctx context.Context, input any, options ...core.Option) (any, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Track visited nodes and node outputs
	visited := make(map[string]bool)
	nodeOutputs := make(map[string]any)

	// Start with entry nodes
	queue := make([]string, 0, len(g.entryNodes))
	for _, entryNode := range g.entryNodes {
		if !visited[entryNode] {
			queue = append(queue, entryNode)
		}
	}

	// Maximum iteration limit as safety net (allows for some redundancy but prevents infinite loops)
	maxIterations := len(g.nodes) * 2
	if maxIterations < 10 {
		maxIterations = 10 // Minimum safety limit
	}
	iterationCount := 0

	// Process nodes in topological order (simplified)
	for len(queue) > 0 {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, iface.ErrTimeout("graph.execute_sequential", ctx.Err())
		default:
		}

		// Safety check: prevent infinite loops
		iterationCount++
		if iterationCount > maxIterations {
			return nil, iface.ErrExecutionFailed("graph.execute_sequential",
				fmt.Errorf("maximum iterations (%d) exceeded, possible cycle detected in graph", maxIterations))
		}

		currentNode := queue[0]
		queue = queue[1:]

		// Skip if already visited (cycle detection)
		if visited[currentNode] {
			continue
		}

		// Execute current node
		nodeCtx := ctx
		var nodeSpan trace.Span
		if g.tracer != nil {
			nodeCtx, nodeSpan = g.tracer.Start(ctx, "graph.node.execute",
				trace.WithAttributes(
					attribute.String("node.name", currentNode),
					attribute.String("node.type", fmt.Sprintf("%T", g.nodes[currentNode])),
				))
			defer nodeSpan.End()
		}
		nodeStart := time.Now()

		var nodeInput any = input
		if len(g.entryNodes) > 1 {
			// For multiple entry nodes, use the original input for all
			nodeInput = input
		}

		output, err := g.nodes[currentNode].Invoke(nodeCtx, nodeInput, options...)
		nodeDuration := time.Since(nodeStart)

		if g.tracer != nil && nodeSpan != nil {
			nodeSpan.SetAttributes(attribute.Float64("node.duration", nodeDuration.Seconds()))
		}

		if err != nil {
			return nil, iface.ErrExecutionFailed("graph.execute_sequential",
				fmt.Errorf("error in graph node %s: %w", currentNode, err))
		}

		nodeOutputs[currentNode] = output
		visited[currentNode] = true

		// Add dependent nodes to queue (only if not already visited to prevent cycles)
		if targets, exists := g.edges[currentNode]; exists {
			for _, target := range targets {
				if !visited[target] {
					queue = append(queue, target)
				}
			}
		}
	}

	// Collect outputs from exit nodes
	if len(g.exitNodes) == 1 {
		if output, exists := nodeOutputs[g.exitNodes[0]]; exists {
			return output, nil
		}
	}

	// For multiple exit nodes, return all outputs
	result := make(map[string]any)
	for _, exitNode := range g.exitNodes {
		if output, exists := nodeOutputs[exitNode]; exists {
			result[exitNode] = output
		}
	}
	return result, nil
}

func (g *BasicGraph) executeParallel(ctx context.Context, input any, options ...core.Option) (any, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// For now, implement as sequential - parallel execution would require
	// more sophisticated dependency resolution
	return g.executeSequential(ctx, input, options...)
}

func (g *BasicGraph) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	var lastErr error

	for i, input := range inputs {
		output, err := g.Invoke(ctx, input, options...)
		if err != nil {
			lastErr = iface.ErrExecutionFailed("graph.batch", fmt.Errorf("error processing batch item %d: %w", i, err))
		}
		results[i] = output
	}

	return results, lastErr
}

func (g *BasicGraph) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	// Basic implementation - stream the first exit node if it supports streaming
	if len(g.exitNodes) == 0 {
		return nil, iface.ErrInvalidConfig("graph.stream", fmt.Errorf("no exit nodes defined"))
	}

	exitNode := g.exitNodes[0]
	if runnable, exists := g.nodes[exitNode]; exists {
		return runnable.Stream(ctx, input, options...)
	}

	return nil, iface.ErrNotFound("graph.stream", fmt.Sprintf("exit node %s", exitNode))
}

func (g *BasicGraph) countEdges() int {
	total := 0
	for _, targets := range g.edges {
		total += len(targets)
	}
	return total
}

// Ensure BasicGraph implements the Graph interface
var _ iface.Graph = (*BasicGraph)(nil)
var _ core.Runnable = (*BasicGraph)(nil)
