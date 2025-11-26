// Package orchestration provides advanced test utilities and comprehensive mocks for testing orchestration implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package orchestration

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// AdvancedMockOrchestrator provides a comprehensive mock implementation for testing.
type AdvancedMockOrchestrator struct {
	lastHealthCheck time.Time
	errorToReturn   error
	nodeResults     map[string]any
	edges           map[string][]string
	mock.Mock
	name             string
	orchestType      string
	healthState      string
	nodes            []string
	responses        []any
	executionOrder   []string
	responseIndex    int
	executionDelay   time.Duration
	callCount        int
	mu               sync.RWMutex
	simulateFailures bool
	shouldError      bool
}

// NewAdvancedMockOrchestrator creates a new advanced mock with configurable behavior.
func NewAdvancedMockOrchestrator(name, orchType string, options ...MockOrchestratorOption) *AdvancedMockOrchestrator {
	mock := &AdvancedMockOrchestrator{
		name:        name,
		orchestType: orchType,
		responses:   []any{},
		nodes:       []string{},
		edges:       make(map[string][]string),
		nodeResults: make(map[string]any),
		healthState: "healthy",
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	return mock
}

// MockOrchestratorOption defines functional options for mock configuration.
type MockOrchestratorOption func(*AdvancedMockOrchestrator)

// WithMockResponses sets predefined responses for the mock.
func WithMockResponses(responses []any) MockOrchestratorOption {
	return func(m *AdvancedMockOrchestrator) {
		m.responses = responses
	}
}

// WithMockError configures the mock to return errors.
func WithMockError(shouldError bool, err error) MockOrchestratorOption {
	return func(m *AdvancedMockOrchestrator) {
		m.shouldError = shouldError
		m.errorToReturn = err
	}
}

// WithExecutionDelay adds artificial delay to mock operations.
func WithExecutionDelay(delay time.Duration) MockOrchestratorOption {
	return func(m *AdvancedMockOrchestrator) {
		m.executionDelay = delay
	}
}

// WithNodes sets the nodes for the mock orchestrator.
func WithNodes(nodes []string) MockOrchestratorOption {
	return func(m *AdvancedMockOrchestrator) {
		m.nodes = nodes
	}
}

// WithEdges sets the edges for the mock orchestrator.
func WithEdges(edges map[string][]string) MockOrchestratorOption {
	return func(m *AdvancedMockOrchestrator) {
		m.edges = edges
	}
}

// Mock implementation methods.
func (m *AdvancedMockOrchestrator) Execute(ctx context.Context, input any) (any, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.executionDelay > 0 {
		time.Sleep(m.executionDelay)
	}

	if m.shouldError {
		return nil, m.errorToReturn
	}

	m.mu.Lock()
	var result any
	if len(m.responses) > m.responseIndex {
		result = m.responses[m.responseIndex]
		m.responseIndex = (m.responseIndex + 1) % len(m.responses)
		m.mu.Unlock()
		return result, nil
	}
	m.mu.Unlock()

	return "mock result for " + m.name, nil
}

func (m *AdvancedMockOrchestrator) ExecuteChain(ctx context.Context, chain iface.Chain) (any, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.executionDelay > 0 {
		time.Sleep(m.executionDelay)
	}

	if m.shouldError {
		return nil, m.errorToReturn
	}

	return "chain execution result for " + m.name, nil
}

func (m *AdvancedMockOrchestrator) ExecuteGraph(ctx context.Context, graph iface.Graph) (any, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.executionDelay > 0 {
		time.Sleep(m.executionDelay)
	}

	if m.shouldError {
		return nil, m.errorToReturn
	}

	return "graph execution result for " + m.name, nil
}

func (m *AdvancedMockOrchestrator) ExecuteWorkflow(ctx context.Context, workflow iface.Workflow) (any, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.executionDelay > 0 {
		time.Sleep(m.executionDelay)
	}

	if m.shouldError {
		return nil, m.errorToReturn
	}

	return "workflow execution result for " + m.name, nil
}

func (m *AdvancedMockOrchestrator) GetName() string {
	return m.name
}

func (m *AdvancedMockOrchestrator) GetType() string {
	return m.orchestType
}

func (m *AdvancedMockOrchestrator) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

func (m *AdvancedMockOrchestrator) CheckHealth() map[string]any {
	m.lastHealthCheck = time.Now()
	return map[string]any{
		"status":       m.healthState,
		"name":         m.name,
		"type":         m.orchestType,
		"call_count":   m.callCount,
		"last_checked": m.lastHealthCheck,
		"nodes_count":  len(m.nodes),
		"edges_count":  len(m.edges),
	}
}

// MockMetricsRecorder provides a mock metrics collector for testing.
type MockMetricsRecorder struct {
	mock.Mock
	recordings []MetricRecord
	mu         sync.RWMutex
}

type MetricRecord struct {
	Timestamp time.Time
	Value     any
	Labels    map[string]string
	Operation string
	Type      string
}

func NewMockMetricsRecorder() *MockMetricsRecorder {
	return &MockMetricsRecorder{
		recordings: make([]MetricRecord, 0),
	}
}

func (m *MockMetricsRecorder) RecordChainExecution(ctx context.Context, duration time.Duration, success bool, chainName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordings = append(m.recordings, MetricRecord{
		Operation: "chain_execution",
		Type:      "duration",
		Value:     duration,
		Labels:    map[string]string{"chain_name": chainName, "success": strconv.FormatBool(success)},
		Timestamp: time.Now(),
	})
}

func (m *MockMetricsRecorder) RecordGraphExecution(ctx context.Context, duration time.Duration, success bool, graphName string, nodeCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordings = append(m.recordings, MetricRecord{
		Operation: "graph_execution",
		Type:      "duration",
		Value:     duration,
		Labels:    map[string]string{"graph_name": graphName, "success": strconv.FormatBool(success), "node_count": strconv.Itoa(nodeCount)},
		Timestamp: time.Now(),
	})
}

func (m *MockMetricsRecorder) RecordWorkflowExecution(ctx context.Context, duration time.Duration, success bool, workflowName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordings = append(m.recordings, MetricRecord{
		Operation: "workflow_execution",
		Type:      "duration",
		Value:     duration,
		Labels:    map[string]string{"workflow_name": workflowName, "success": strconv.FormatBool(success)},
		Timestamp: time.Now(),
	})
}

func (m *MockMetricsRecorder) GetRecordings() []MetricRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]MetricRecord, len(m.recordings))
	copy(result, m.recordings)
	return result
}

func (m *MockMetricsRecorder) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordings = m.recordings[:0]
}

// Test data creation helpers

// CreateTestChain creates a test chain configuration.
func CreateTestChain(name string, steps []string) iface.Chain {
	return &TestChain{
		name:  name,
		steps: steps,
	}
}

type TestChain struct {
	name  string
	steps []string
}

func (c *TestChain) GetName() string    { return c.name }
func (c *TestChain) GetSteps() []string { return c.steps }
func (c *TestChain) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return fmt.Sprintf("executed chain %s with %d steps", c.name, len(c.steps)), nil
}

func (c *TestChain) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i := range inputs {
		results[i] = fmt.Sprintf("batch result %d for %s", i, c.name)
	}
	return results, nil
}

func (c *TestChain) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		ch <- "stream result for chain " + c.name
	}()
	return ch, nil
}
func (c *TestChain) GetInputKeys() []string        { return []string{"input"} }
func (c *TestChain) GetOutputKeys() []string       { return []string{"output"} }
func (c *TestChain) GetMemory() memoryiface.Memory { return nil }

// CreateTestGraph creates a test graph configuration.
func CreateTestGraph(name string, nodes []string, edges map[string][]string) iface.Graph {
	return &TestGraph{
		name:  name,
		nodes: nodes,
		edges: edges,
	}
}

type TestGraph struct {
	edges map[string][]string
	name  string
	nodes []string
}

func (g *TestGraph) GetName() string               { return g.name }
func (g *TestGraph) GetNodes() []string            { return g.nodes }
func (g *TestGraph) GetEdges() map[string][]string { return g.edges }
func (g *TestGraph) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return fmt.Sprintf("executed graph %s with %d nodes", g.name, len(g.nodes)), nil
}

func (g *TestGraph) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i := range inputs {
		results[i] = fmt.Sprintf("batch result %d for graph %s", i, g.name)
	}
	return results, nil
}

func (g *TestGraph) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		ch <- "stream result for graph " + g.name
	}()
	return ch, nil
}

func (g *TestGraph) AddNode(name string, runnable core.Runnable) error {
	g.nodes = append(g.nodes, name)
	return nil
}

func (g *TestGraph) AddEdge(sourceNode, targetNode string) error {
	if g.edges == nil {
		g.edges = make(map[string][]string)
	}
	g.edges[sourceNode] = append(g.edges[sourceNode], targetNode)
	return nil
}
func (g *TestGraph) SetEntryPoint(nodeNames []string) error  { return nil }
func (g *TestGraph) SetFinishPoint(nodeNames []string) error { return nil }

// CreateTestWorkflow creates a test workflow configuration.
func CreateTestWorkflow(name string, tasks []string) iface.Workflow {
	return &TestWorkflow{
		name:  name,
		tasks: tasks,
	}
}

type TestWorkflow struct {
	name  string
	tasks []string
}

func (w *TestWorkflow) GetName() string    { return w.name }
func (w *TestWorkflow) GetTasks() []string { return w.tasks }
func (w *TestWorkflow) Execute(ctx context.Context, input any) (string, string, error) {
	workflowID := fmt.Sprintf("workflow-%s-%d", w.name, time.Now().UnixNano())
	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())
	return workflowID, runID, nil
}

func (w *TestWorkflow) GetResult(ctx context.Context, workflowID, runID string) (any, error) {
	return fmt.Sprintf("result for workflow %s (run: %s)", workflowID, runID), nil
}

func (w *TestWorkflow) Signal(ctx context.Context, workflowID, runID, signalName string, data any) error {
	return nil
}

func (w *TestWorkflow) Query(ctx context.Context, workflowID, runID, queryType string, args ...any) (any, error) {
	return "query result for " + queryType, nil
}

func (w *TestWorkflow) Cancel(ctx context.Context, workflowID, runID string) error {
	return nil
}

func (w *TestWorkflow) Terminate(ctx context.Context, workflowID, runID, reason string, details ...any) error {
	return nil
}

// Assertion helpers

// AssertHealthCheck validates health check results.
func AssertHealthCheck(t *testing.T, health map[string]any, expectedStatus string) {
	assert.Contains(t, health, "status")
	assert.Equal(t, expectedStatus, health["status"])
	assert.Contains(t, health, "name")
	assert.Contains(t, health, "type")
}

// AssertExecutionResult validates execution results.
func AssertExecutionResult(t *testing.T, result any, expectedPattern string) {
	t.Helper()
	assert.NotNil(t, result)
	if str, ok := result.(string); ok {
		assert.Contains(t, str, expectedPattern)
	}
}

// AssertErrorType validates error types and codes.
func AssertErrorType(t *testing.T, err error, expectedCode string) {
	t.Helper()
	require.Error(t, err)
	// Add specific orchestration error type checking if available
}

// Performance testing helpers

// ConcurrentTestRunner runs tests concurrently for performance testing.
type ConcurrentTestRunner struct {
	testFunc      func() error
	NumGoroutines int
	TestDuration  time.Duration
}

func NewConcurrentTestRunner(numGoroutines int, duration time.Duration, testFunc func() error) *ConcurrentTestRunner {
	return &ConcurrentTestRunner{
		NumGoroutines: numGoroutines,
		TestDuration:  duration,
		testFunc:      testFunc,
	}
}

func (r *ConcurrentTestRunner) Run() error {
	var wg sync.WaitGroup
	errChan := make(chan error, r.NumGoroutines)
	stopChan := make(chan struct{})

	// Start timer
	timer := time.AfterFunc(r.TestDuration, func() {
		close(stopChan)
	})
	defer timer.Stop()

	// Start worker goroutines
	for i := 0; i < r.NumGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					if err := r.testFunc(); err != nil {
						errChan <- err
						return
					}
				}
			}
		}()
	}

	// Wait for completion
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// RunLoadTest executes a load test scenario.
func RunLoadTest(t *testing.T, orchestrator *AdvancedMockOrchestrator, numRequests, concurrency int) {
	t.Helper()
	var wg sync.WaitGroup
	errChan := make(chan error, numRequests)

	semaphore := make(chan struct{}, concurrency)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			ctx := context.Background()
			_, err := orchestrator.Execute(ctx, fmt.Sprintf("request-%d", requestID))
			if err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verify no errors occurred
	for err := range errChan {
		require.NoError(t, err)
	}

	// Verify expected call count
	assert.Equal(t, numRequests, orchestrator.GetCallCount())
}

// Integration test helpers

// IntegrationTestHelper provides utilities for integration testing.
type IntegrationTestHelper struct {
	orchestrators map[string]*AdvancedMockOrchestrator
	metrics       *MockMetricsRecorder
}

func NewIntegrationTestHelper() *IntegrationTestHelper {
	return &IntegrationTestHelper{
		orchestrators: make(map[string]*AdvancedMockOrchestrator),
		metrics:       NewMockMetricsRecorder(),
	}
}

func (h *IntegrationTestHelper) AddOrchestrator(name string, orch *AdvancedMockOrchestrator) {
	h.orchestrators[name] = orch
}

func (h *IntegrationTestHelper) GetOrchestrator(name string) *AdvancedMockOrchestrator {
	return h.orchestrators[name]
}

func (h *IntegrationTestHelper) GetMetrics() *MockMetricsRecorder {
	return h.metrics
}

func (h *IntegrationTestHelper) Reset() {
	for _, orch := range h.orchestrators {
		orch.responseIndex = 0
		orch.callCount = 0
	}
	h.metrics.Clear()
}
