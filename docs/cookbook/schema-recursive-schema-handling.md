---
title: "Recursive Schema Handling for Graphs"
package: "schema"
category: "validation"
complexity: "advanced"
---

# Recursive Schema Handling for Graphs

## Problem

You need to validate and process schema structures that contain recursive references, such as agent-to-agent communication graphs, nested tool call chains, or hierarchical document structures where nodes reference each other.

## Solution

Implement a recursive validation visitor that tracks visited nodes to prevent infinite loops and validates the entire graph structure. This works because graph structures require cycle detection and depth-limited traversal to ensure both correctness and termination.

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
    
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

var tracer = otel.Tracer("beluga.schema.graph")

// GraphNode represents a node in a recursive schema graph
type GraphNode struct {
    ID          string
    Data        interface{}
    References  []string // IDs of referenced nodes
    Visited     bool
}

// GraphValidator validates recursive schema structures
type GraphValidator struct {
    nodes       map[string]*GraphNode
    maxDepth    int
    visited     map[string]bool
    currentPath []string
}

// NewGraphValidator creates a new graph validator
func NewGraphValidator(maxDepth int) *GraphValidator {
    return &GraphValidator{
        nodes:    make(map[string]*GraphNode),
        maxDepth: maxDepth,
        visited:  make(map[string]bool),
    }
}

// AddNode adds a node to the graph
func (gv *GraphValidator) AddNode(id string, data interface{}, refs []string) {
    gv.nodes[id] = &GraphNode{
        ID:         id,
        Data:       data,
        References: refs,
    }
}

// ValidateGraph validates the entire graph structure
func (gv *GraphValidator) ValidateGraph(ctx context.Context, startID string) error {
    ctx, span := tracer.Start(ctx, "validator.validate_graph")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("graph.start_node", startID),
        attribute.Int("graph.max_depth", gv.maxDepth),
        attribute.Int("graph.total_nodes", len(gv.nodes)),
    )
    
    // Reset state
    gv.visited = make(map[string]bool)
    gv.currentPath = []string{}
    
    // Validate starting from root
    if err := gv.validateNode(ctx, startID, 0); err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return fmt.Errorf("graph validation failed: %w", err)
    }
    
    // Check for unreachable nodes
    unreachable := gv.findUnreachableNodes(startID)
    if len(unreachable) > 0 {
        span.SetAttributes(attribute.StringSlice("graph.unreachable", unreachable))
        log.Printf("Warning: %d unreachable nodes found", len(unreachable))
    }
    
    span.SetStatus(trace.StatusOK, "graph validation passed")
    return nil
}

// validateNode recursively validates a node and its references
func (gv *GraphValidator) validateNode(ctx context.Context, nodeID string, depth int) error {
    // Check depth limit
    if depth > gv.maxDepth {
        return fmt.Errorf("maximum depth %d exceeded at node %s", gv.maxDepth, nodeID)
    }
    
    // Check for cycles
    for _, pathID := range gv.currentPath {
        if pathID == nodeID {
            return fmt.Errorf("cycle detected: %v -> %s", gv.currentPath, nodeID)
        }
    }
    
    // Get node
    node, exists := gv.nodes[nodeID]
    if !exists {
        return fmt.Errorf("node %s not found", nodeID)
    }
    
    // Mark as visited
    gv.visited[nodeID] = true
    gv.currentPath = append(gv.currentPath, nodeID)
    defer func() {
        gv.currentPath = gv.currentPath[:len(gv.currentPath)-1]
    }()
    
    // Validate node data (example: validate as message if applicable)
    if msg, ok := node.Data.(schema.Message); ok {
        if err := gv.validateMessage(ctx, msg); err != nil {
            return fmt.Errorf("node %s validation failed: %w", nodeID, err)
        }
    }
    
    // Recursively validate references
    for _, refID := range node.References {
        if err := gv.validateNode(ctx, refID, depth+1); err != nil {
            return err
        }
    }
    
    return nil
}

// validateMessage validates a message within the graph context
func (gv *GraphValidator) validateMessage(ctx context.Context, msg schema.Message) error {
    // Basic validation
    if msg == nil {
        return fmt.Errorf("message is nil")
    }
    if msg.GetContent() == "" {
        return fmt.Errorf("message content is empty")
    }
    return nil
}

// findUnreachableNodes finds nodes not reachable from start
func (gv *GraphValidator) findUnreachableNodes(startID string) []string {
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

    // Create graph validator
    validator := NewGraphValidator(10) // Max depth of 10
    
    // Build a graph structure (e.g., agent communication chain)
    validator.AddNode("agent1", schema.NewHumanMessage("Start"), []string{"agent2"})
    validator.AddNode("agent2", schema.NewAIMessage("Response"), []string{"agent3"})
    validator.AddNode("agent3", schema.NewHumanMessage("Follow-up"), []string{}) // Leaf node
    
    // Validate graph
    if err := validator.ValidateGraph(ctx, "agent1"); err != nil {
        log.Fatalf("Graph validation failed: %v", err)
    }
    fmt.Println("Graph validated successfully")
}
```

## Explanation

Let's break down what's happening:

1. **Cycle detection** - Notice how we track `currentPath` to detect cycles. When we encounter a node ID that's already in the current path, we've found a cycle. This prevents infinite loops in recursive structures.

2. **Depth limiting** - We enforce a maximum depth to prevent stack overflow and ensure termination. This is important because malicious or malformed graphs could have extremely deep nesting.

3. **Visited tracking** - We maintain a `visited` map to identify unreachable nodes. This helps detect disconnected graph components that might indicate data integrity issues.

```go
**Key insight:** Always implement cycle detection and depth limits when processing recursive structures. Without these safeguards, you risk infinite loops or stack overflows.

## Testing

```
Here's how to test this solution:
```go
func TestGraphValidator_CycleDetection(t *testing.T) {
    ctx := context.Background()
    validator := NewGraphValidator(10)
    
    // Create a cycle: A -> B -> C -> A
    validator.AddNode("A", schema.NewHumanMessage("A"), []string{"B"})
    validator.AddNode("B", schema.NewAIMessage("B"), []string{"C"})
    validator.AddNode("C", schema.NewHumanMessage("C"), []string{"A"})
    
    err := validator.ValidateGraph(ctx, "A")
    if err == nil {
        t.Error("Expected cycle detection to fail")
    }
    if !strings.Contains(err.Error(), "cycle detected") {
        t.Errorf("Expected cycle error, got: %v", err)
    }
}

func TestGraphValidator_DepthLimit(t *testing.T) {
    ctx := context.Background()
    validator := NewGraphValidator(3) // Low depth limit

    // Create deep chain
    validator.AddNode("1", schema.NewHumanMessage("1"), []string{"2"})
    validator.AddNode("2", schema.NewAIMessage("2"), []string{"3"})
    validator.AddNode("3", schema.NewHumanMessage("3"), []string{"4"})
    validator.AddNode("4", schema.NewAIMessage("4"), []string{"5"})
    validator.AddNode("5", schema.NewHumanMessage("5"), []string{})
    
    err := validator.ValidateGraph(ctx, "1")
    if err == nil {
        t.Error("Expected depth limit to be exceeded")
    }
}
## Variations

### Parallel Validation

For large graphs, validate branches in parallel:
func (gv *GraphValidator) validateNodeParallel(ctx context.Context, nodeID string, depth int) error {
    node := gv.nodes[nodeID]
    
    // Validate references in parallel
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
### Graph Serialization

Serialize validated graphs for persistence:
```go
func (gv *GraphValidator) Serialize() ([]byte, error) {
    return json.Marshal(gv.nodes)
}
```

## Related Recipes

- **[Schema Custom Validation Middleware](./schema-custom-validation-middleware.md)** - Apply custom validation rules
- **[Orchestration Parallel Node Execution](./orchestration-parallel-node-execution.md)** - Execute graph nodes in parallel
- **[Schema Package Guide](../package_design_patterns.md)** - For a deeper understanding of schema structures
