// Package agents provides advanced test utilities and comprehensive mocks for testing agent implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package agents

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockAgent provides a comprehensive mock implementation for testing
type AdvancedMockAgent struct {
	mock.Mock

	// Configuration
	name      string
	agentType string
	callCount int
	mu        sync.RWMutex

	// Configurable behavior
	shouldError      bool
	errorToReturn    error
	responses        []interface{}
	responseIndex    int
	executionDelay   time.Duration
	simulateFailures bool

	// Agent-specific data
	tools            []tools.Tool
	executionHistory []ExecutionRecord
	state            map[string]interface{}
	planningSteps    []string

	// Health check data
	healthState     string
	lastHealthCheck time.Time
}

// ExecutionRecord tracks agent execution history for testing
type ExecutionRecord struct {
	Input     interface{}
	Output    interface{}
	Error     error
	Duration  time.Duration
	Timestamp time.Time
}

// NewAdvancedMockAgent creates a new advanced mock with configurable behavior
func NewAdvancedMockAgent(name, agentType string, options ...MockAgentOption) *AdvancedMockAgent {
	mock := &AdvancedMockAgent{
		name:             name,
		agentType:        agentType,
		responses:        []interface{}{},
		tools:            []tools.Tool{},
		executionHistory: make([]ExecutionRecord, 0),
		state:            make(map[string]interface{}),
		planningSteps:    []string{},
		healthState:      "healthy",
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	return mock
}

// MockAgentOption defines functional options for mock configuration
type MockAgentOption func(*AdvancedMockAgent)

// WithMockError configures the mock to return errors
func WithMockError(shouldError bool, err error) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.shouldError = shouldError
		a.errorToReturn = err
	}
}

// WithMockResponses sets predefined responses for the mock
func WithMockResponses(responses []interface{}) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.responses = responses
	}
}

// WithExecutionDelay adds artificial delay to mock operations
func WithExecutionDelay(delay time.Duration) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.executionDelay = delay
	}
}

// WithMockTools sets the tools available to the agent
func WithMockTools(agentTools []tools.Tool) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.tools = agentTools
	}
}

// WithPlanningSteps sets planning steps for the mock agent
func WithPlanningSteps(steps []string) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.planningSteps = steps
	}
}

// WithAgentState sets initial state for the mock agent
func WithAgentState(state map[string]interface{}) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.state = make(map[string]interface{})
		for k, v := range state {
			a.state[k] = v
		}
	}
}

// Mock implementation methods for core.Runnable interface
func (a *AdvancedMockAgent) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return a.execute(ctx, input)
}

func (a *AdvancedMockAgent) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := a.execute(ctx, input)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (a *AdvancedMockAgent) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		result, err := a.execute(ctx, input)
		if err != nil {
			// In a real implementation, would send error through channel
			return
		}
		ch <- result
	}()
	return ch, nil
}

// Private execute method used by Runnable implementations
func (a *AdvancedMockAgent) execute(ctx context.Context, input any) (any, error) {
	a.mu.Lock()
	a.callCount++
	start := time.Now()
	a.mu.Unlock()

	if a.executionDelay > 0 {
		time.Sleep(a.executionDelay)
	}

	duration := time.Since(start)
	var result any
	var err error

	if a.shouldError {
		err = a.errorToReturn
	} else {
		if len(a.responses) > a.responseIndex {
			result = a.responses[a.responseIndex]
			a.responseIndex = (a.responseIndex + 1) % len(a.responses)
		} else {
			result = fmt.Sprintf("Agent %s executed with input: %v", a.name, input)
		}
	}

	// Record execution
	a.mu.Lock()
	a.executionHistory = append(a.executionHistory, ExecutionRecord{
		Input:     input,
		Output:    result,
		Error:     err,
		Duration:  duration,
		Timestamp: time.Now(),
	})
	a.mu.Unlock()

	return result, err
}

// Mock implementation methods for Agent interface
func (a *AdvancedMockAgent) Plan(ctx context.Context, intermediateSteps []iface.IntermediateStep, inputs map[string]any) (iface.AgentAction, iface.AgentFinish, error) {
	a.mu.Lock()
	a.callCount++
	a.mu.Unlock()

	if a.executionDelay > 0 {
		time.Sleep(a.executionDelay)
	}

	if a.shouldError {
		return iface.AgentAction{}, iface.AgentFinish{}, a.errorToReturn
	}

	// Simple planning: if we have steps to execute, return an action
	if len(a.planningSteps) > 0 && len(intermediateSteps) < len(a.planningSteps) {
		stepIndex := len(intermediateSteps)
		return iface.AgentAction{
			Tool:      a.planningSteps[stepIndex],
			ToolInput: fmt.Sprintf("input for step %d", stepIndex+1),
			Log:       fmt.Sprintf("Planning step %d: %s", stepIndex+1, a.planningSteps[stepIndex]),
		}, iface.AgentFinish{}, nil
	}

	// Finish execution
	return iface.AgentAction{}, iface.AgentFinish{
		ReturnValues: map[string]any{"result": fmt.Sprintf("Agent %s completed planning", a.name)},
		Log:          "Planning completed successfully",
	}, nil
}

func (a *AdvancedMockAgent) InputVariables() []string {
	return []string{"input"}
}

func (a *AdvancedMockAgent) OutputVariables() []string {
	return []string{"output"}
}

func (a *AdvancedMockAgent) GetTools() []tools.Tool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]tools.Tool, len(a.tools))
	copy(result, a.tools)
	return result
}

func (a *AdvancedMockAgent) GetConfig() schema.AgentConfig {
	// Return a mock config
	return schema.AgentConfig{}
}

func (a *AdvancedMockAgent) GetLLM() llmsiface.LLM {
	// Return nil for mock - in real implementation would return actual LLM
	return nil
}

func (a *AdvancedMockAgent) GetMetrics() iface.MetricsRecorder {
	// Return nil for mock - in real implementation would return metrics recorder
	return nil
}

// LifecycleManager interface implementation
func (a *AdvancedMockAgent) Initialize(config map[string]interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Update state from config
	for k, v := range config {
		a.state[k] = v
	}

	return nil
}

func (a *AdvancedMockAgent) Execute() error {
	// Basic execute without context for lifecycle interface
	_, err := a.execute(context.Background(), "lifecycle_execute")
	return err
}

func (a *AdvancedMockAgent) Shutdown() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.healthState = "shutdown"
	return nil
}

func (a *AdvancedMockAgent) GetState() iface.AgentState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return iface.StateReady // Default to ready state
}

// EventEmitter interface implementation
func (a *AdvancedMockAgent) RegisterEventHandler(eventType string, handler iface.EventHandler) {
	// Mock implementation - in real implementation would store handlers
}

func (a *AdvancedMockAgent) EmitEvent(eventType string, payload interface{}) {
	// Mock implementation - in real implementation would call handlers
}

// HealthChecker interface implementation
func (a *AdvancedMockAgent) CheckHealth() map[string]interface{} {
	a.lastHealthCheck = time.Now()
	return map[string]interface{}{
		"status":          a.healthState,
		"name":            a.name,
		"type":            a.agentType,
		"call_count":      a.callCount,
		"execution_count": len(a.executionHistory),
		"tools_count":     len(a.tools),
		"state_keys":      len(a.state),
		"last_checked":    a.lastHealthCheck,
	}
}

// Additional helper methods for testing
func (a *AdvancedMockAgent) GetName() string {
	return a.name
}

func (a *AdvancedMockAgent) GetType() string {
	return a.agentType
}

func (a *AdvancedMockAgent) GetInternalState() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make(map[string]interface{})
	for k, v := range a.state {
		result[k] = v
	}
	return result
}

func (a *AdvancedMockAgent) SetInternalState(key string, value interface{}) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state[key] = value
}

func (a *AdvancedMockAgent) GetCallCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.callCount
}

func (a *AdvancedMockAgent) GetExecutionHistory() []ExecutionRecord {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]ExecutionRecord, len(a.executionHistory))
	copy(result, a.executionHistory)
	return result
}

// MockTool provides a simple mock tool for testing
type MockTool struct {
	tools.BaseTool
	callCount int
	mu        sync.RWMutex
}

func NewMockTool(name, description string) *MockTool {
	tool := &MockTool{}
	tool.SetName(name)
	tool.SetDescription(description)
	tool.SetInputSchema(map[string]interface{}{
		"type":        "string",
		"description": "Tool input",
	})
	return tool
}

func (t *MockTool) Execute(ctx context.Context, input any) (any, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.callCount++

	return fmt.Sprintf("Tool %s executed with input: %v", t.Name(), input), nil
}

func (t *MockTool) GetCallCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.callCount
}

// Test data creation helpers

// CreateTestAgentConfig creates a test agent configuration
func CreateTestAgentConfig(agentType string) Config {
	return Config{
		DefaultMaxRetries:    3,
		DefaultRetryDelay:    time.Second,
		DefaultTimeout:       30 * time.Second,
		DefaultMaxIterations: 10,
		EnableMetrics:        true,
		EnableTracing:        true,
		MetricsPrefix:        "test_agents",
		TracingServiceName:   "test-agents",
		ExecutorConfig: ExecutorConfig{
			DefaultMaxConcurrency:   5,
			HandleParsingErrors:     true,
			ReturnIntermediateSteps: false,
			MaxConcurrentExecutions: 50,
			ExecutionTimeout:        5 * time.Minute,
		},
		AgentConfigs: make(map[string]schema.AgentConfig),
	}
}

// CreateTestTools creates a set of mock tools for testing
func CreateTestTools(count int) []tools.Tool {
	testTools := make([]tools.Tool, count)
	for i := 0; i < count; i++ {
		toolName := fmt.Sprintf("test_tool_%d", i+1)
		description := fmt.Sprintf("Test tool %d for agent testing", i+1)
		testTools[i] = NewMockTool(toolName, description)
	}
	return testTools
}

// Assertion helpers

// AssertAgentExecution validates agent execution results
func AssertAgentExecution(t *testing.T, result interface{}, expectedPattern string) {
	assert.NotNil(t, result)
	if str, ok := result.(string); ok {
		assert.Contains(t, str, expectedPattern)
	}
}

// AssertPlanningResult validates agent planning results
func AssertPlanningResult(t *testing.T, steps []string, expectedMinSteps int) {
	assert.GreaterOrEqual(t, len(steps), expectedMinSteps)
	for i, step := range steps {
		assert.NotEmpty(t, step, "Planning step %d should not be empty", i+1)
	}
}

// AssertAgentHealth validates agent health check results
func AssertAgentHealth(t *testing.T, health map[string]interface{}, expectedStatus string) {
	assert.Contains(t, health, "status")
	assert.Equal(t, expectedStatus, health["status"])
	assert.Contains(t, health, "name")
	assert.Contains(t, health, "type")
	assert.Contains(t, health, "call_count")
}

// AssertErrorType validates error types and codes
func AssertErrorType(t *testing.T, err error, expectedCode string) {
	assert.Error(t, err)
	var agentErr *AgentError
	if assert.ErrorAs(t, err, &agentErr) {
		assert.Equal(t, expectedCode, agentErr.Code)
	}
}

// Performance testing helpers

// ConcurrentTestRunner runs agent tests concurrently for performance testing
type ConcurrentTestRunner struct {
	NumGoroutines int
	TestDuration  time.Duration
	testFunc      func() error
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

// RunLoadTest executes a load test scenario on agents
func RunLoadTest(t *testing.T, agent *AdvancedMockAgent, numOperations int, concurrency int) {
	var wg sync.WaitGroup
	errChan := make(chan error, numOperations)

	semaphore := make(chan struct{}, concurrency)

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(opID int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			ctx := context.Background()

			if opID%2 == 0 {
				// Test Invoke (Execute)
				_, err := agent.Invoke(ctx, fmt.Sprintf("operation-%d", opID))
				if err != nil {
					errChan <- err
				}
			} else {
				// Test Plan
				inputs := map[string]any{"input": fmt.Sprintf("planning-%d", opID)}
				_, _, err := agent.Plan(ctx, []iface.IntermediateStep{}, inputs)
				if err != nil {
					errChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verify no errors occurred
	for err := range errChan {
		assert.NoError(t, err)
	}

	// Verify expected call count
	assert.Equal(t, numOperations, agent.GetCallCount())
}

// Integration test helpers

// IntegrationTestHelper provides utilities for integration testing
type IntegrationTestHelper struct {
	agents   map[string]*AdvancedMockAgent
	tools    map[string]*MockTool
	executor *MockExecutor
}

func NewIntegrationTestHelper() *IntegrationTestHelper {
	return &IntegrationTestHelper{
		agents:   make(map[string]*AdvancedMockAgent),
		tools:    make(map[string]*MockTool),
		executor: NewMockExecutor(),
	}
}

func (h *IntegrationTestHelper) AddAgent(name string, agent *AdvancedMockAgent) {
	h.agents[name] = agent
}

func (h *IntegrationTestHelper) AddTool(name string, tool *MockTool) {
	h.tools[name] = tool
}

func (h *IntegrationTestHelper) GetAgent(name string) *AdvancedMockAgent {
	return h.agents[name]
}

func (h *IntegrationTestHelper) GetTool(name string) *MockTool {
	return h.tools[name]
}

func (h *IntegrationTestHelper) GetExecutor() *MockExecutor {
	return h.executor
}

func (h *IntegrationTestHelper) Reset() {
	for _, agent := range h.agents {
		agent.callCount = 0
		agent.responseIndex = 0
		agent.executionHistory = make([]ExecutionRecord, 0)
	}
	for _, tool := range h.tools {
		tool.callCount = 0
	}
	if h.executor != nil {
		h.executor.Reset()
	}
}

// MockExecutor provides a mock executor for testing
type MockExecutor struct {
	executions    []ExecutionRecord
	callCount     int
	mu            sync.RWMutex
	shouldError   bool
	errorToReturn error
}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		executions: make([]ExecutionRecord, 0),
	}
}

func (e *MockExecutor) Execute(ctx context.Context, agent iface.CompositeAgent, input interface{}) (interface{}, error) {
	e.mu.Lock()
	e.callCount++
	start := time.Now()
	e.mu.Unlock()

	if e.shouldError {
		return nil, e.errorToReturn
	}

	// Execute the agent using Runnable interface
	result, err := agent.Invoke(ctx, input)
	duration := time.Since(start)

	// Record execution
	e.mu.Lock()
	e.executions = append(e.executions, ExecutionRecord{
		Input:     input,
		Output:    result,
		Error:     err,
		Duration:  duration,
		Timestamp: time.Now(),
	})
	e.mu.Unlock()

	return result, err
}

func (e *MockExecutor) GetExecutions() []ExecutionRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]ExecutionRecord, len(e.executions))
	copy(result, e.executions)
	return result
}

func (e *MockExecutor) GetCallCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.callCount
}

func (e *MockExecutor) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.executions = make([]ExecutionRecord, 0)
	e.callCount = 0
}

func (e *MockExecutor) WithError(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.shouldError = true
	e.errorToReturn = err
}

// AgentScenarioRunner runs common agent scenarios
type AgentScenarioRunner struct {
	agent    iface.CompositeAgent
	executor *MockExecutor
}

func NewAgentScenarioRunner(agent iface.CompositeAgent) *AgentScenarioRunner {
	return &AgentScenarioRunner{
		agent:    agent,
		executor: NewMockExecutor(),
	}
}

func (r *AgentScenarioRunner) RunTaskExecutionScenario(ctx context.Context, tasks []string) error {
	for i, task := range tasks {
		result, err := r.executor.Execute(ctx, r.agent, task)
		if err != nil {
			return fmt.Errorf("task %d failed: %w", i+1, err)
		}

		if result == nil {
			return fmt.Errorf("task %d returned nil result", i+1)
		}
	}

	return nil
}

func (r *AgentScenarioRunner) RunPlanningScenario(ctx context.Context, problems []string) ([][]string, error) {
	plans := make([][]string, len(problems))

	for i, problem := range problems {
		// Use the planning interface from Agent
		inputs := map[string]any{"input": problem}
		action, finish, err := r.agent.Plan(ctx, []iface.IntermediateStep{}, inputs)

		if err != nil {
			return nil, fmt.Errorf("planning for problem %d failed: %w", i+1, err)
		}

		// Extract plan from action or finish
		if action.Tool != "" {
			plans[i] = []string{action.Tool}
		} else if len(finish.ReturnValues) > 0 {
			plans[i] = []string{"finished"}
		} else {
			plans[i] = []string{}
		}
	}

	return plans, nil
}

func (r *AgentScenarioRunner) RunToolUsageScenario(ctx context.Context, toolTasks []string) error {
	agentTools := r.agent.GetTools()

	for i := range toolTasks {
		// Simulate using tools for the task
		for j, tool := range agentTools {
			_, err := tool.Execute(ctx, fmt.Sprintf("task-%d-tool-%d", i+1, j+1))
			if err != nil {
				return fmt.Errorf("tool execution failed for task %d: %w", i+1, err)
			}
		}
	}

	return nil
}

// Test configuration helpers

// CreateTestAgentWithTools creates an agent configured with test tools
func CreateTestAgentWithTools(name string, toolCount int) *AdvancedMockAgent {
	testTools := CreateTestTools(toolCount)
	return NewAdvancedMockAgent(name, "base", WithMockTools(testTools))
}

// CreateTestAgentWithState creates an agent with predefined state
func CreateTestAgentWithState(name string, state map[string]interface{}) *AdvancedMockAgent {
	return NewAdvancedMockAgent(name, "base", WithAgentState(state))
}

// CreateTestExecutionPlan creates a test execution plan
func CreateTestExecutionPlan(steps int) []string {
	plan := make([]string, steps)
	for i := 0; i < steps; i++ {
		plan[i] = fmt.Sprintf("execute_step_%d", i+1)
	}
	return plan
}

// CreateCollaborativeAgents creates agents configured for collaboration
func CreateCollaborativeAgents(count int, sharedState map[string]interface{}) []*AdvancedMockAgent {
	agents := make([]*AdvancedMockAgent, count)

	for i := 0; i < count; i++ {
		agentName := fmt.Sprintf("collaborative_agent_%d", i+1)
		agentType := "collaborative"

		// Create shared tools for collaboration
		collaborativeTools := []tools.Tool{
			NewMockTool("communicate", "Communicate with other agents"),
			NewMockTool("coordinate", "Coordinate task execution"),
			NewMockTool("share_state", "Share state information"),
		}

		agents[i] = NewAdvancedMockAgent(agentName, agentType,
			WithMockTools(collaborativeTools),
			WithAgentState(sharedState),
		)
	}

	return agents
}

// Benchmarking helpers

// AgentBenchmark provides benchmarking utilities for agents
type AgentBenchmark struct {
	agent iface.CompositeAgent
	tasks []string
}

func NewAgentBenchmark(agent iface.CompositeAgent, taskCount int) *AgentBenchmark {
	tasks := make([]string, taskCount)
	for i := 0; i < taskCount; i++ {
		tasks[i] = fmt.Sprintf("benchmark_task_%d", i+1)
	}

	return &AgentBenchmark{
		agent: agent,
		tasks: tasks,
	}
}

func (b *AgentBenchmark) BenchmarkExecution(iterations int) (time.Duration, error) {
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		task := b.tasks[i%len(b.tasks)]
		_, err := b.agent.Invoke(ctx, task)
		if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}

func (b *AgentBenchmark) BenchmarkPlanning(iterations int) (time.Duration, error) {
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		inputs := map[string]any{"input": b.tasks[i%len(b.tasks)]}
		_, _, err := b.agent.Plan(ctx, []iface.IntermediateStep{}, inputs)
		if err != nil {
			return 0, err
		}
	}
	return time.Since(start), nil
}
