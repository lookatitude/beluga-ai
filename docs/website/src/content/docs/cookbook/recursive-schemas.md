---
title: "Recursive Schema Handling"
description: "Validate and process recursive schema structures with cycle detection and depth limiting."
---

## Problem

You need to validate and process schema structures that contain recursive references, such as agent-to-agent communication graphs, nested tool call chains, or hierarchical document structures where nodes reference each other.

## Solution

Implement a recursive validation visitor that tracks visited nodes to prevent infinite loops and validates the entire graph structure. Graph structures require cycle detection and depth-limited traversal to ensure both correctness and termination.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.schema.graph")

// GraphNode represents a node in a recursive schema graph.
type GraphNode struct {
	ID         string
	Data       interface{}
	References []string // IDs of referenced nodes
}

// GraphValidator validates recursive schema structures.
type GraphValidator struct {
	nodes       map[string]*GraphNode
	maxDepth    int
	visited     map[string]bool
	currentPath []string
}

// NewGraphValidator creates a new graph validator.
func NewGraphValidator(maxDepth int) *GraphValidator {
	return &GraphValidator{
		nodes:    make(map[string]*GraphNode),
		maxDepth: maxDepth,
		visited:  make(map[string]bool),
	}
}

// AddNode adds a node to the graph.
func (gv *GraphValidator) AddNode(id string, data interface{}, refs []string) {
	gv.nodes[id] = &GraphNode{
		ID:         id,
		Data:       data,
		References: refs,
	}
}

// ValidateGraph validates the entire graph structure.
func (gv *GraphValidator) ValidateGraph(ctx context.Context, startID string) error {
	ctx, span := tracer.Start(ctx, "validator.validate_graph")
	defer span.End()

	span.SetAttributes(
		attribute.String("graph.start_node", startID),
		attribute.Int("graph.max_depth", gv.maxDepth),
		attribute.Int("graph.total_nodes", len(gv.nodes)),
	)

	gv.visited = make(map[string]bool)
	gv.currentPath = []string{}

	if err := gv.validateNode(ctx, startID, 0); err != nil {
		span.RecordError(err)
		span.SetStatus(trace.StatusError, err.Error())
		return fmt.Errorf("graph validation failed: %w", err)
	}

	unreachable := gv.findUnreachableNodes()
	if len(unreachable) > 0 {
		span.SetAttributes(attribute.StringSlice("graph.unreachable", unreachable))
		log.Printf("Warning: %d unreachable nodes found", len(unreachable))
	}

	span.SetStatus(trace.StatusOK, "graph validation passed")
	return nil
}

// validateNode recursively validates a node and its references.
func (gv *GraphValidator) validateNode(ctx context.Context, nodeID string, depth int) error {
	if depth > gv.maxDepth {
		return fmt.Errorf("maximum depth %d exceeded at node %s", gv.maxDepth, nodeID)
	}

	for _, pathID := range gv.currentPath {
		if pathID == nodeID {
			return fmt.Errorf("cycle detected: %v -> %s", gv.currentPath, nodeID)
		}
	}

	node, exists := gv.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	gv.visited[nodeID] = true
	gv.currentPath = append(gv.currentPath, nodeID)
	defer func() {
		gv.currentPath = gv.currentPath[:len(gv.currentPath)-1]
	}()

	if msg, ok := node.Data.(schema.Message); ok {
		if err := gv.validateMessage(ctx, msg); err != nil {
			return fmt.Errorf("node %s validation failed: %w", nodeID, err)
		}
	}

	for _, refID := range node.References {
		if err := gv.validateNode(ctx, refID, depth+1); err != nil {
			return err
		}
	}

	return nil
}

func (gv *GraphValidator) validateMessage(ctx context.Context, msg schema.Message) error {
	if msg == nil {
		return fmt.Errorf("message is nil")
	}
	if msg.GetContent() == "" {
		return fmt.Errorf("message content is empty")
	}
	return nil
}

func (gv *GraphValidator) findUnreachableNodes() []string {
	unreachable := []string{}
	for id := range gv.nodes {
		if !gv.visited[id] {
			unreachable = append(unreachable, id)
		}
	}
	return unreachable
}

func main() {
	ctx := context.Background()

	validator := NewGraphValidator(10)

	validator.AddNode("agent1", schema.NewHumanMessage("Start"), []string{"agent2"})
	validator.AddNode("agent2", schema.NewAIMessage("Response"), []string{"agent3"})
	validator.AddNode("agent3", schema.NewHumanMessage("Follow-up"), []string{})

	if err := validator.ValidateGraph(ctx, "agent1"); err != nil {
		log.Fatalf("Graph validation failed: %v", err)
	}
	fmt.Println("Graph validated successfully")
}
```

## Explanation

1. **Cycle detection** -- The validator tracks `currentPath` during traversal. When a node ID already appears in the current path, a cycle has been found. This prevents infinite loops in recursive structures.

2. **Depth limiting** -- A maximum depth is enforced to prevent stack overflow and ensure termination. Malformed or adversarial graphs could have extremely deep nesting without this guard.

3. **Visited tracking** -- A `visited` map identifies unreachable nodes after traversal. Disconnected graph components may indicate data integrity issues.

**Key insight:** Always implement cycle detection and depth limits when processing recursive structures. Without these safeguards, you risk infinite loops or stack overflows.

## Variations

### Parallel Validation

For large graphs, validate independent branches concurrently:

```go
func (gv *GraphValidator) validateNodeParallel(ctx context.Context, nodeID string, depth int) error {
	node := gv.nodes[nodeID]

	var wg sync.WaitGroup
	errCh := make(chan error, len(node.References))

	for _, refID := range node.References {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if err := gv.validateNode(ctx, id, depth+1); err != nil {
				errCh <- err
			}
		}(refID)
	}

	wg.Wait()
	close(errCh)

	if err := <-errCh; err != nil {
		return err
	}
	return nil
}
```

## Related Recipes

- **[Schema Validation Middleware](./schema-validation)** -- Apply custom validation rules
- **[Parallel Node Execution](./parallel-nodes)** -- Execute graph nodes in parallel
