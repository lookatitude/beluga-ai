package graph

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRunnable is a mock implementation of core.Runnable for testing
type MockRunnable struct {
	mock.Mock
	name string
}

func (m *MockRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	args := m.Called(ctx, input, opts)
	return args.Get(0), args.Error(1)
}

func (m *MockRunnable) Batch(ctx context.Context, inputs []any, opts ...core.Option) ([]any, error) {
	args := m.Called(ctx, inputs, opts)
	return args.Get(0).([]any), args.Error(1)
}

func (m *MockRunnable) Stream(ctx context.Context, input any, opts ...core.Option) (<-chan any, error) {
	args := m.Called(ctx, input, opts)
	return args.Get(0).(<-chan any), args.Error(1)
}

// MockTracer is a mock implementation of trace.Tracer for testing
type MockTracer struct {
	mock.Mock
}

func (m *MockTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	args := m.Called(ctx, spanName, opts)
	return args.Get(0).(context.Context), args.Get(1).(trace.Span)
}

// MockSpan is a mock implementation of trace.Span
type MockSpan struct {
	mock.Mock
}

func (m *MockSpan) End(options ...trace.SpanEndOption) {
	m.Called(options)
}

func (m *MockSpan) AddEvent(name string, options ...trace.EventOption) {
	m.Called(name, options)
}

func (m *MockSpan) IsRecording() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSpan) RecordError(err error, options ...trace.EventOption) {
	m.Called(err, options)
}

func (m *MockSpan) SpanContext() trace.SpanContext {
	args := m.Called()
	return args.Get(0).(trace.SpanContext)
}

func (m *MockSpan) SetStatus(code codes.Code, description string) {
	m.Called(code, description)
}

func (m *MockSpan) SetName(name string) {
	m.Called(name)
}

func (m *MockSpan) SetAttributes(kv ...attribute.KeyValue) {
	m.Called(kv)
}

func (m *MockSpan) TracerProvider() trace.TracerProvider {
	args := m.Called()
	return args.Get(0).(trace.TracerProvider)
}

func TestNewBasicGraph(t *testing.T) {
	config := iface.GraphConfig{
		Name:        "test-graph",
		EntryPoints: []string{"start"},
		ExitPoints:  []string{"end"},
	}

	graph := NewBasicGraph(config, nil)

	assert.NotNil(t, graph)
	assert.Equal(t, config, graph.config)
	assert.NotNil(t, graph.nodes)
	assert.NotNil(t, graph.edges)
	assert.Equal(t, []string{"start"}, graph.entryNodes)
	assert.Equal(t, []string{"end"}, graph.exitNodes)
}

func TestBasicGraph_AddNode(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{}, nil)

	runnable := &MockRunnable{name: "test-node"}

	// Test successful addition
	err := graph.AddNode("node1", runnable)
	assert.NoError(t, err)
	assert.Contains(t, graph.nodes, "node1")
	assert.Equal(t, runnable, graph.nodes["node1"])

	// Test duplicate node error
	err = graph.AddNode("node1", runnable)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestBasicGraph_AddEdge(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{}, nil)

	runnable1 := &MockRunnable{name: "node1"}
	runnable2 := &MockRunnable{name: "node2"}

	// Add nodes first
	err := graph.AddNode("node1", runnable1)
	require.NoError(t, err)
	err = graph.AddNode("node2", runnable2)
	require.NoError(t, err)

	// Test successful edge addition
	err = graph.AddEdge("node1", "node2")
	assert.NoError(t, err)
	assert.Contains(t, graph.edges, "node1")
	assert.Contains(t, graph.edges["node1"], "node2")

	// Test edge to non-existent source node
	err = graph.AddEdge("nonexistent", "node2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source node nonexistent")

	// Test edge to non-existent target node
	err = graph.AddEdge("node1", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target node nonexistent")
}

func TestBasicGraph_SetEntryPoint(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{}, nil)

	runnable := &MockRunnable{name: "entry-node"}
	err := graph.AddNode("entry", runnable)
	require.NoError(t, err)

	// Test successful entry point setting
	err = graph.SetEntryPoint([]string{"entry"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"entry"}, graph.entryNodes)

	// Test setting non-existent entry point
	err = graph.SetEntryPoint([]string{"nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "entry node nonexistent")
}

func TestBasicGraph_SetFinishPoint(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{}, nil)

	runnable := &MockRunnable{name: "exit-node"}
	err := graph.AddNode("exit", runnable)
	require.NoError(t, err)

	// Test successful finish point setting
	err = graph.SetFinishPoint([]string{"exit"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"exit"}, graph.exitNodes)

	// Test setting non-existent finish point
	err = graph.SetFinishPoint([]string{"nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exit node nonexistent")
}

func TestBasicGraph_Invoke_LinearGraph(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{Name: "linear-graph"}, nil)

	// Setup nodes
	node1 := &MockRunnable{name: "node1"}
	node2 := &MockRunnable{name: "node2"}

	node1.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"output1": "result1"}, nil)
	node2.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"output2": "result2"}, nil)

	// Build graph: node1 -> node2
	err := graph.AddNode("node1", node1)
	require.NoError(t, err)
	err = graph.AddNode("node2", node2)
	require.NoError(t, err)
	err = graph.AddEdge("node1", "node2")
	require.NoError(t, err)
	err = graph.SetEntryPoint([]string{"node1"})
	require.NoError(t, err)
	err = graph.SetFinishPoint([]string{"node2"})
	require.NoError(t, err)

	// Execute graph
	input := map[string]any{"input": "test"}
	result, err := graph.Invoke(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, map[string]any{"output2": "result2"}, result)

	node1.AssertExpectations(t)
	node2.AssertExpectations(t)
}

func TestBasicGraph_Invoke_BranchingGraph(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{Name: "branching-graph"}, nil)

	// Setup nodes
	startNode := &MockRunnable{name: "start"}
	branch1Node := &MockRunnable{name: "branch1"}
	branch2Node := &MockRunnable{name: "branch2"}

	startNode.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"start": "done"}, nil)
	branch1Node.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"branch1": "result1"}, nil)
	branch2Node.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"branch2": "result2"}, nil)

	// Build graph: start -> [branch1, branch2]
	err := graph.AddNode("start", startNode)
	require.NoError(t, err)
	err = graph.AddNode("branch1", branch1Node)
	require.NoError(t, err)
	err = graph.AddNode("branch2", branch2Node)
	require.NoError(t, err)
	err = graph.AddEdge("start", "branch1")
	require.NoError(t, err)
	err = graph.AddEdge("start", "branch2")
	require.NoError(t, err)
	err = graph.SetEntryPoint([]string{"start"})
	require.NoError(t, err)
	err = graph.SetFinishPoint([]string{"branch1", "branch2"})
	require.NoError(t, err)

	// Execute graph
	input := map[string]any{"input": "test"}
	result, err := graph.Invoke(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Should return results from both exit nodes
	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, resultMap, "branch1")
	assert.Contains(t, resultMap, "branch2")

	startNode.AssertExpectations(t)
	branch1Node.AssertExpectations(t)
	branch2Node.AssertExpectations(t)
}

func TestBasicGraph_Invoke_WithTimeout(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{
		Name:    "timeout-graph",
		Timeout: 1, // 1 second
	}, nil)

	node := &MockRunnable{name: "slow-node"}
	node.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"result": "done"}, nil)

	err := graph.AddNode("node1", node)
	require.NoError(t, err)
	err = graph.SetEntryPoint([]string{"node1"})
	require.NoError(t, err)
	err = graph.SetFinishPoint([]string{"node1"})
	require.NoError(t, err)

	input := map[string]any{"input": "test"}
	result, err := graph.Invoke(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	node.AssertExpectations(t)
}

func TestBasicGraph_Invoke_NodeError(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{Name: "error-graph"}, nil)

	failingNode := &MockRunnable{name: "failing-node"}
	failingNode.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(nil, errors.New("node failed"))

	err := graph.AddNode("failing", failingNode)
	require.NoError(t, err)
	err = graph.SetEntryPoint([]string{"failing"})
	require.NoError(t, err)
	err = graph.SetFinishPoint([]string{"failing"})
	require.NoError(t, err)

	input := map[string]any{"input": "test"}
	result, err := graph.Invoke(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "error in graph node failing")

	failingNode.AssertExpectations(t)
}

func TestBasicGraph_Invoke_ParallelExecution(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{
		Name:                    "parallel-graph",
		EnableParallelExecution: true,
	}, nil)

	node1 := &MockRunnable{name: "node1"}
	node2 := &MockRunnable{name: "node2"}

	node1.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"result1": "done"}, nil)
	node2.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"result2": "done"}, nil)

	err := graph.AddNode("node1", node1)
	require.NoError(t, err)
	err = graph.AddNode("node2", node2)
	require.NoError(t, err)
	err = graph.SetEntryPoint([]string{"node1", "node2"})
	require.NoError(t, err)
	err = graph.SetFinishPoint([]string{"node1", "node2"})
	require.NoError(t, err)

	input := map[string]any{"input": "test"}
	result, err := graph.Invoke(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Should return results from both exit nodes
	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, resultMap, "node1")
	assert.Contains(t, resultMap, "node2")

	node1.AssertExpectations(t)
	node2.AssertExpectations(t)
}

func TestBasicGraph_Batch(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{Name: "batch-graph"}, nil)

	node := &MockRunnable{name: "batch-node"}
	node.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"result": "batch_done"}, nil).Times(3)

	err := graph.AddNode("node", node)
	require.NoError(t, err)
	err = graph.SetEntryPoint([]string{"node"})
	require.NoError(t, err)
	err = graph.SetFinishPoint([]string{"node"})
	require.NoError(t, err)

	inputs := []any{
		map[string]any{"input": "test1"},
		map[string]any{"input": "test2"},
		map[string]any{"input": "test3"},
	}

	results, err := graph.Batch(context.Background(), inputs)

	assert.NoError(t, err)
	assert.Len(t, results, 3)
	for _, result := range results {
		assert.NotNil(t, result)
	}

	node.AssertExpectations(t)
}

func TestBasicGraph_Batch_WithErrors(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{Name: "batch-error-graph"}, nil)

	node := &MockRunnable{name: "batch-node"}
	node.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).
		Return(nil, errors.New("batch node failed")).Times(1)
	node.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).
		Return(map[string]any{"result": "success"}, nil).Times(2)

	err := graph.AddNode("node", node)
	require.NoError(t, err)
	err = graph.SetEntryPoint([]string{"node"})
	require.NoError(t, err)
	err = graph.SetFinishPoint([]string{"node"})
	require.NoError(t, err)

	inputs := []any{
		map[string]any{"input": "test1"},
		map[string]any{"input": "test2"},
		map[string]any{"input": "test3"},
	}

	results, err := graph.Batch(context.Background(), inputs)

	assert.Error(t, err)
	assert.Len(t, results, 3)
	assert.Contains(t, err.Error(), "error processing batch item")

	node.AssertExpectations(t)
}

func TestBasicGraph_Stream_Success(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{Name: "stream-graph"}, nil)

	streamNode := &MockRunnable{name: "stream-node"}
	streamChan := make(chan any, 1)
	streamChan <- "stream_result"
	close(streamChan)
	streamNode.On("Stream", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return((<-chan any)(streamChan), nil)

	err := graph.AddNode("stream", streamNode)
	require.NoError(t, err)
	err = graph.SetFinishPoint([]string{"stream"})
	require.NoError(t, err)

	input := map[string]any{"input": "test"}
	resultChan, err := graph.Stream(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, resultChan)

	// Read from the stream
	select {
	case result := <-resultChan:
		assert.Equal(t, "stream_result", result)
	case <-time.After(1 * time.Second):
		t.Fatal("Expected result from stream")
	}

	streamNode.AssertExpectations(t)
}

func TestBasicGraph_Stream_NoExitNodes(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{Name: "no-exit-graph"}, nil)

	input := map[string]any{"input": "test"}
	resultChan, err := graph.Stream(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, resultChan)
	assert.Contains(t, err.Error(), "no exit nodes defined")
}

func TestBasicGraph_Stream_ExitNodeNotFound(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{Name: "missing-exit-graph"}, nil)

	err := graph.SetFinishPoint([]string{"nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exit node nonexistent not found")

	// If SetFinishPoint fails, we can't test Stream
	if err != nil {
		return
	}

	input := map[string]any{"input": "test"}
	resultChan, err := graph.Stream(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, resultChan)
}

func TestBasicGraph_countEdges(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{}, nil)

	runnable1 := &MockRunnable{name: "node1"}
	runnable2 := &MockRunnable{name: "node2"}
	runnable3 := &MockRunnable{name: "node3"}

	// Add nodes
	err := graph.AddNode("node1", runnable1)
	require.NoError(t, err)
	err = graph.AddNode("node2", runnable2)
	require.NoError(t, err)
	err = graph.AddNode("node3", runnable3)
	require.NoError(t, err)

	// Add edges: node1 -> [node2, node3], node2 -> node3
	err = graph.AddEdge("node1", "node2")
	require.NoError(t, err)
	err = graph.AddEdge("node1", "node3")
	require.NoError(t, err)
	err = graph.AddEdge("node2", "node3")
	require.NoError(t, err)

	// Count should be 3
	count := graph.countEdges()
	assert.Equal(t, 3, count)
}

func TestBasicGraph_ComplexTopology(t *testing.T) {
	graph := NewBasicGraph(iface.GraphConfig{Name: "complex-graph"}, nil)

	// Create a diamond topology: A -> B, A -> C, B -> D, C -> D
	nodeA := &MockRunnable{name: "nodeA"}
	nodeB := &MockRunnable{name: "nodeB"}
	nodeC := &MockRunnable{name: "nodeC"}
	nodeD := &MockRunnable{name: "nodeD"}

	nodeA.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"A": "done"}, nil)
	nodeB.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"B": "done"}, nil)
	nodeC.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"C": "done"}, nil)
	nodeD.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"D": "final"}, nil)

	// Build the diamond topology
	err := graph.AddNode("A", nodeA)
	require.NoError(t, err)
	err = graph.AddNode("B", nodeB)
	require.NoError(t, err)
	err = graph.AddNode("C", nodeC)
	require.NoError(t, err)
	err = graph.AddNode("D", nodeD)
	require.NoError(t, err)

	err = graph.AddEdge("A", "B")
	require.NoError(t, err)
	err = graph.AddEdge("A", "C")
	require.NoError(t, err)
	err = graph.AddEdge("B", "D")
	require.NoError(t, err)
	err = graph.AddEdge("C", "D")
	require.NoError(t, err)

	err = graph.SetEntryPoint([]string{"A"})
	require.NoError(t, err)
	err = graph.SetFinishPoint([]string{"D"})
	require.NoError(t, err)

	// Execute the graph
	input := map[string]any{"input": "diamond"}
	result, err := graph.Invoke(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, map[string]any{"D": "final"}, result)

	// Verify all nodes were called
	nodeA.AssertExpectations(t)
	nodeB.AssertExpectations(t)
	nodeC.AssertExpectations(t)
	nodeD.AssertExpectations(t)
}
