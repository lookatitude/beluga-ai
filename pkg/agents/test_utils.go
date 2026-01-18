// Package agents provides advanced test utilities and comprehensive mocks for testing agent implementations.
// This file contains utilities designed to support both unit tests and integration tests.
//
// Test Coverage Exclusions:
//
// The following code paths are intentionally excluded from 100% coverage requirements:
//
// 1. Panic Recovery Paths:
//    - Panic handlers in concurrent test runners (ConcurrentTestRunner, ConcurrentStreamingTestRunner)
//    - These paths are difficult to test without causing actual panics in test code
//
// 2. Context Cancellation Edge Cases:
//    - Some context cancellation paths in streaming operations are difficult to reliably test
//    - Race conditions between context cancellation and channel operations
//
// 3. Error Paths Requiring System Conditions:
//    - Network errors that require actual network failures
//    - File system errors that require specific OS conditions
//    - Memory exhaustion scenarios
//
// 4. Provider-Specific Untestable Paths:
//    - Some provider implementations have paths that require external service failures
//    - These are tested through integration tests rather than unit tests
//
// 5. Test Utility Functions:
//    - Helper functions in test_utils.go that are used by tests but not directly tested
//    - These are validated through their usage in actual test cases
//
// 6. Initialization Code:
//    - Package init() functions and global variable initialization
//    - These are executed automatically and difficult to test in isolation
//
// All exclusions are documented here to maintain transparency about coverage goals.
// The target is 100% coverage of testable code paths, excluding the above categories.
package agents

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	"github.com/stretchr/testify/require"
)

// Static error variables for testing (err113 compliance)
var (
	errMockStreamingError    = errors.New("mock streaming error")
	errTaskReturnedNilResult = errors.New("task returned nil result")
)

// AdvancedMockAgent provides a comprehensive mock implementation for testing.
type AdvancedMockAgent struct {
	lastHealthCheck time.Time
	errorToReturn   error
	state           map[string]any
	mock.Mock
	name             string
	agentType        string
	healthState      string
	tools            []tools.Tool
	planningSteps    []string
	responses        []any
	executionHistory []ExecutionRecord
	responseIndex    int
	executionDelay   time.Duration
	callCount        int
	mu               sync.RWMutex
	simulateFailures bool
	shouldError      bool
}

// ExecutionRecord tracks agent execution history for testing.
type ExecutionRecord struct {
	Timestamp time.Time
	Input     any
	Output    any
	Error     error
	Duration  time.Duration
}

// NewAdvancedMockAgent creates a new advanced mock with configurable behavior.
func NewAdvancedMockAgent(name, agentType string, options ...MockAgentOption) *AdvancedMockAgent {
	mock := &AdvancedMockAgent{
		name:             name,
		agentType:        agentType,
		responses:        []any{},
		tools:            []tools.Tool{},
		executionHistory: make([]ExecutionRecord, 0),
		state:            make(map[string]any),
		planningSteps:    []string{},
		healthState:      "healthy",
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	return mock
}

// MockAgentOption defines functional options for mock configuration.
type MockAgentOption func(*AdvancedMockAgent)

// WithMockError configures the mock to return errors.
func WithMockError(shouldError bool, err error) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.shouldError = shouldError
		a.errorToReturn = err
	}
}

// WithMockAgentError configures the mock to return an AgentError with the specified code.
func WithMockAgentError(op, code string, underlyingErr error) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.shouldError = true
		a.errorToReturn = NewAgentError(op, code, underlyingErr)
	}
}

// WithMockExecutionError configures the mock to return an ExecutionError.
func WithMockExecutionError(agent string, step int, action string, underlyingErr error, retryable bool) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.shouldError = true
		a.errorToReturn = NewExecutionError(agent, step, action, underlyingErr, retryable)
	}
}

// WithMockPlanningError configures the mock to return a PlanningError.
func WithMockPlanningError(agent string, inputKeys []string, underlyingErr error) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.shouldError = true
		a.errorToReturn = NewPlanningError(agent, inputKeys, underlyingErr)
	}
}

// WithMockStreamingError configures the mock to return a StreamingError.
func WithMockStreamingError(op, agent, code string, underlyingErr error) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.shouldError = true
		a.errorToReturn = NewStreamingError(op, agent, code, underlyingErr)
	}
}

// WithMockValidationError configures the mock to return a ValidationError.
func WithMockValidationError(field, message string) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.shouldError = true
		a.errorToReturn = NewValidationError(field, message)
	}
}

// WithMockFactoryError configures the mock to return a FactoryError.
func WithMockFactoryError(agentType string, config any, underlyingErr error) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.shouldError = true
		a.errorToReturn = NewFactoryError(agentType, config, underlyingErr)
	}
}

// WithMockErrorCode configures the mock to return an AgentError with a specific error code.
// This is a convenience function for common error codes.
func WithMockErrorCode(code string, underlyingErr error) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.shouldError = true
		a.errorToReturn = NewAgentError("mock_operation", code, underlyingErr)
	}
}

// WithMockResponses sets predefined responses for the mock.
func WithMockResponses(responses []any) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.responses = responses
	}
}

// WithExecutionDelay adds artificial delay to mock operations.
func WithExecutionDelay(delay time.Duration) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.executionDelay = delay
	}
}

// WithMockTools sets the tools available to the agent.
func WithMockTools(agentTools []tools.Tool) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.tools = agentTools
	}
}

// WithPlanningSteps sets planning steps for the mock agent.
func WithPlanningSteps(steps []string) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.planningSteps = steps
	}
}

// WithAgentState sets initial state for the mock agent.
func WithAgentState(state map[string]any) MockAgentOption {
	return func(a *AdvancedMockAgent) {
		a.state = make(map[string]any)
		for k, v := range state {
			a.state[k] = v
		}
	}
}

// Mock implementation methods for core.Runnable interface.
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

// Private execute method used by Runnable implementations.
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

// Mock implementation methods for Agent interface.
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

// LifecycleManager interface implementation.
func (a *AdvancedMockAgent) Initialize(config map[string]any) error {
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

// EventEmitter interface implementation.
func (a *AdvancedMockAgent) RegisterEventHandler(eventType string, handler iface.EventHandler) {
	// Mock implementation - in real implementation would store handlers
}

func (a *AdvancedMockAgent) EmitEvent(eventType string, payload any) {
	// Mock implementation - in real implementation would call handlers
}

// HealthChecker interface implementation.
func (a *AdvancedMockAgent) CheckHealth() map[string]any {
	a.lastHealthCheck = time.Now()
	return map[string]any{
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

// Additional helper methods for testing.
func (a *AdvancedMockAgent) GetName() string {
	return a.name
}

func (a *AdvancedMockAgent) GetType() string {
	return a.agentType
}

func (a *AdvancedMockAgent) GetInternalState() map[string]any {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make(map[string]any)
	for k, v := range a.state {
		result[k] = v
	}
	return result
}

func (a *AdvancedMockAgent) SetInternalState(key string, value any) {
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

// MockTool provides a simple mock tool for testing.
type MockTool struct {
	tools.BaseTool
	callCount int
	mu        sync.RWMutex
}

func NewMockTool(name, description string) *MockTool {
	tool := &MockTool{
		BaseTool: tools.BaseTool{},
	}
	tool.SetName(name)
	tool.SetDescription(description)
	tool.SetInputSchema(map[string]any{
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

// CreateTestAgentConfig creates a test agent configuration.
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

// CreateTestTools creates a set of mock tools for testing.
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

// AssertAgentExecution validates agent execution results.
func AssertAgentExecution(t *testing.T, result any, expectedPattern string) {
	assert.NotNil(t, result)
	if str, ok := result.(string); ok {
		assert.Contains(t, str, expectedPattern)
	}
}

// AssertPlanningResult validates agent planning results.
func AssertPlanningResult(t *testing.T, steps []string, expectedMinSteps int) {
	assert.GreaterOrEqual(t, len(steps), expectedMinSteps)
	for i, step := range steps {
		assert.NotEmpty(t, step, "Planning step %d should not be empty", i+1)
	}
}

// AssertAgentHealth validates agent health check results.
func AssertAgentHealth(t *testing.T, health map[string]any, expectedStatus string) {
	assert.Contains(t, health, "status")
	assert.Equal(t, expectedStatus, health["status"])
	assert.Contains(t, health, "name")
	assert.Contains(t, health, "type")
	assert.Contains(t, health, "call_count")
}

// AssertErrorType validates error types and codes.
func AssertErrorType(t *testing.T, err error, expectedCode string) {
	require.Error(t, err)
	var agentErr *AgentError
	if assert.ErrorAs(t, err, &agentErr) {
		assert.Equal(t, expectedCode, agentErr.Code)
	}
}

// Performance testing helpers

// ConcurrentTestRunner runs agent tests concurrently for performance testing.
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
	var stopOnce sync.Once

	// Ensure stopChan is always closed
	stopFunc := func() {
		stopOnce.Do(func() {
			close(stopChan)
		})
	}

	// Create context with timeout as additional safety mechanism
	ctx, cancel := context.WithTimeout(context.Background(), r.TestDuration+time.Second)
	defer cancel()

	// Start timer
	timer := time.AfterFunc(r.TestDuration, stopFunc)
	defer func() {
		timer.Stop()
		stopFunc() // Ensure stopChan is closed even if timer didn't fire
	}()

	// Start worker goroutines
	for i := 0; i < r.NumGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				case <-ctx.Done():
					return
				default:
					if err := r.testFunc(); err != nil {
						select {
						case errChan <- err:
						default:
							// Channel full, but we still want to stop
						}
						stopFunc() // Signal other goroutines to stop
						return
					}
				}
			}
		}()
	}

	// Wait for completion with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines completed
	case <-ctx.Done():
		// Timeout exceeded, force stop
		stopFunc()
		wg.Wait() // Wait for goroutines to exit
	}

	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// RunLoadTest executes a load test scenario on agents.
func RunLoadTest(t *testing.T, agent *AdvancedMockAgent, numOperations, concurrency int) {
	t.Helper()
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
		require.NoError(t, err)
	}

	// Verify expected call count
	assert.Equal(t, numOperations, agent.GetCallCount())
}

// Integration test helpers

// IntegrationTestHelper provides utilities for integration testing.
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

// MockExecutor provides a mock executor for testing.
type MockExecutor struct {
	errorToReturn error
	executions    []ExecutionRecord
	callCount     int
	mu            sync.RWMutex
	shouldError   bool
}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		executions: make([]ExecutionRecord, 0),
	}
}

func (e *MockExecutor) Execute(ctx context.Context, agent iface.CompositeAgent, input any) (any, error) {
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

// AgentScenarioRunner runs common agent scenarios.
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
			return fmt.Errorf("task %d: %w", i+1, errTaskReturnedNilResult)
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

// CreateTestAgentWithTools creates an agent configured with test tools.
func CreateTestAgentWithTools(name string, toolCount int) *AdvancedMockAgent {
	testTools := CreateTestTools(toolCount)
	return NewAdvancedMockAgent(name, "base", WithMockTools(testTools))
}

// CreateTestAgentWithState creates an agent with predefined state.
func CreateTestAgentWithState(name string, state map[string]any) *AdvancedMockAgent {
	return NewAdvancedMockAgent(name, "base", WithAgentState(state))
}

// CreateTestExecutionPlan creates a test execution plan.
func CreateTestExecutionPlan(steps int) []string {
	plan := make([]string, steps)
	for i := 0; i < steps; i++ {
		plan[i] = fmt.Sprintf("execute_step_%d", i+1)
	}
	return plan
}

// CreateCollaborativeAgents creates agents configured for collaboration.
func CreateCollaborativeAgents(count int, sharedState map[string]any) []*AdvancedMockAgent {
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

// AgentBenchmark provides benchmarking utilities for agents.
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

// Streaming Agent Test Utilities
// These utilities support the StreamingAgent interface for testing.

// Note: AgentStreamChunk is now defined in pkg/agents/iface/streaming_agent.go
// We use iface.AgentStreamChunk in the mock implementations below.

// AdvancedMockStreamingAgent provides a comprehensive mock implementation for streaming agents.
// This struct extends AdvancedMockAgent with streaming capabilities.
type AdvancedMockStreamingAgent struct {
	streamingError error
	*AdvancedMockAgent
	chunkGenerator      func() iface.AgentStreamChunk
	streamingChunks     []iface.AgentStreamChunk
	streamingDelay      time.Duration
	streamCount         int
	mu                  sync.RWMutex
	shouldErrorOnStream bool
}

// MockStreamingAgentOption defines functional options for configuring streaming agent mocks.
type MockStreamingAgentOption func(*AdvancedMockStreamingAgent)

// WithMockStreamingChunks sets predefined chunks to stream.
func WithMockStreamingChunks(chunks []iface.AgentStreamChunk) MockStreamingAgentOption {
	return func(a *AdvancedMockStreamingAgent) {
		a.streamingChunks = chunks
	}
}

// WithStreamingDelay adds artificial delay between streaming chunks.
func WithStreamingDelay(delay time.Duration) MockStreamingAgentOption {
	return func(a *AdvancedMockStreamingAgent) {
		a.streamingDelay = delay
	}
}

// WithStreamingError configures the mock to return an error during streaming.
func WithStreamingError(shouldError bool, err error) MockStreamingAgentOption {
	return func(a *AdvancedMockStreamingAgent) {
		a.shouldErrorOnStream = shouldError
		a.streamingError = err
	}
}

// WithChunkGenerator sets a custom function to generate chunks dynamically.
func WithChunkGenerator(generator func() iface.AgentStreamChunk) MockStreamingAgentOption {
	return func(a *AdvancedMockStreamingAgent) {
		a.chunkGenerator = generator
	}
}

// NewAdvancedMockStreamingAgent creates a new advanced streaming agent mock.
func NewAdvancedMockStreamingAgent(baseAgent *AdvancedMockAgent, options ...MockStreamingAgentOption) *AdvancedMockStreamingAgent {
	mock := &AdvancedMockStreamingAgent{
		AdvancedMockAgent: baseAgent,
		streamingChunks:   []iface.AgentStreamChunk{},
		streamingDelay:    10 * time.Millisecond,
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	return mock
}

// StreamExecute implements the StreamingAgent interface.
// This method simulates streaming execution by returning a channel of chunks.
func (a *AdvancedMockStreamingAgent) StreamExecute(ctx context.Context, inputs map[string]any) (<-chan iface.AgentStreamChunk, error) {
	a.mu.Lock()
	a.streamCount++
	a.mu.Unlock()

	if a.shouldErrorOnStream {
		if a.streamingError != nil {
			return nil, a.streamingError
		}
		return nil, errMockStreamingError
	}

	ch := make(chan iface.AgentStreamChunk, 10)

	go func() {
		defer close(ch)

		// Use predefined chunks if available
		if len(a.streamingChunks) > 0 {
			for _, chunk := range a.streamingChunks {
				select {
				case <-ctx.Done():
					ch <- iface.AgentStreamChunk{Err: ctx.Err()}
					return
				case ch <- chunk:
				}

				if chunk.Err != nil || chunk.Finish != nil {
					return
				}

				if a.streamingDelay > 0 {
					select {
					case <-ctx.Done():
						ch <- iface.AgentStreamChunk{Err: ctx.Err()}
						return
					case <-time.After(a.streamingDelay):
					}
				}
			}
			return
		}

		// Default: generate simple response chunks
		response := fmt.Sprintf("Agent %s executed with input: %v", a.name, inputs)
		words := strings.Fields(response)
		for i, word := range words {
			select {
			case <-ctx.Done():
				ch <- iface.AgentStreamChunk{Err: ctx.Err()}
				return
			case ch <- iface.AgentStreamChunk{
				Content: word + " ",
				Metadata: map[string]any{
					"chunk_index": i,
					"timestamp":   time.Now(),
				},
			}:
			}

			if a.streamingDelay > 0 && i < len(words)-1 {
				select {
				case <-ctx.Done():
					ch <- iface.AgentStreamChunk{Err: ctx.Err()}
					return
				case <-time.After(a.streamingDelay):
				}
			}
		}

		// Send final chunk with finish
		select {
		case <-ctx.Done():
			ch <- iface.AgentStreamChunk{Err: ctx.Err()}
			return
		case ch <- iface.AgentStreamChunk{
			Finish: &iface.AgentFinish{
				ReturnValues: map[string]any{"output": response},
				Log:          "Streaming execution completed",
			},
		}:
		}
	}()

	return ch, nil
}

// StreamPlan implements the StreamingAgent interface.
// This method simulates streaming planning by returning a channel of chunks.
func (a *AdvancedMockStreamingAgent) StreamPlan(ctx context.Context, intermediateSteps []iface.IntermediateStep, inputs map[string]any) (<-chan iface.AgentStreamChunk, error) {
	a.mu.Lock()
	a.streamCount++
	a.mu.Unlock()

	if a.shouldErrorOnStream {
		if a.streamingError != nil {
			return nil, a.streamingError
		}
		return nil, errMockStreamingError
	}

	ch := make(chan iface.AgentStreamChunk, 10)

	go func() {
		defer close(ch)

		// Simulate planning with streaming responses
		planLog := fmt.Sprintf("Planning step %d", len(intermediateSteps)+1)
		select {
		case <-ctx.Done():
			ch <- iface.AgentStreamChunk{Err: ctx.Err()}
			return
		case ch <- iface.AgentStreamChunk{
			Content: planLog + " ",
			Metadata: map[string]any{
				"step_index": len(intermediateSteps),
				"timestamp":  time.Now(),
			},
		}:
		}

		if a.streamingDelay > 0 {
			select {
			case <-ctx.Done():
				ch <- iface.AgentStreamChunk{Err: ctx.Err()}
				return
			case <-time.After(a.streamingDelay):
			}
		}

		// Send final chunk with action or finish
		if len(a.planningSteps) > 0 && len(intermediateSteps) < len(a.planningSteps) {
			stepIndex := len(intermediateSteps)
			select {
			case <-ctx.Done():
				ch <- iface.AgentStreamChunk{Err: ctx.Err()}
				return
			case ch <- iface.AgentStreamChunk{
				Action: &iface.AgentAction{
					Tool:      a.planningSteps[stepIndex],
					ToolInput: fmt.Sprintf("input for step %d", stepIndex+1),
					Log:       fmt.Sprintf("Planning step %d: %s", stepIndex+1, a.planningSteps[stepIndex]),
				},
			}:
			}
		} else {
			select {
			case <-ctx.Done():
				ch <- iface.AgentStreamChunk{Err: ctx.Err()}
				return
			case ch <- iface.AgentStreamChunk{
				Finish: &iface.AgentFinish{
					ReturnValues: map[string]any{"result": fmt.Sprintf("Agent %s completed planning", a.name)},
					Log:          "Planning completed successfully",
				},
			}:
			}
		}
	}()

	return ch, nil
}

// GetStreamCount returns the number of streams started.
func (a *AdvancedMockStreamingAgent) GetStreamCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.streamCount
}

// MockStreamingExecutor provides a mock executor with streaming capabilities.
type MockStreamingExecutor struct {
	streamingError error
	*MockExecutor
	streamingChunks     []ExecutionChunk
	streamingDelay      time.Duration
	mu                  sync.RWMutex
	shouldErrorOnStream bool
}

// ExecutionChunk represents a chunk of execution output.
// This type matches the contract definition for streaming executor chunks.
type ExecutionChunk struct {
	Timestamp   time.Time
	Err         error
	ToolResult  *ToolExecutionResult
	FinalAnswer *schema.FinalAnswer
	Step        schema.Step
	Content     string
}

// ToolExecutionResult represents the result of tool execution.
type ToolExecutionResult struct {
	Err      error
	Input    map[string]any
	Output   map[string]any
	ToolName string
	Duration time.Duration
}

// NewMockStreamingExecutor creates a new mock streaming executor.
func NewMockStreamingExecutor(baseExecutor *MockExecutor) *MockStreamingExecutor {
	return &MockStreamingExecutor{
		MockExecutor:    baseExecutor,
		streamingChunks: []ExecutionChunk{},
		streamingDelay:  10 * time.Millisecond,
	}
}

// ExecuteStreamingPlan implements the StreamingExecutor interface (to be defined in Phase 3.4).
func (e *MockStreamingExecutor) ExecuteStreamingPlan(ctx context.Context, agent iface.Agent, plan []schema.Step) (<-chan ExecutionChunk, error) {
	if len(plan) == 0 {
		return nil, errors.New("plan cannot be empty")
	}

	ch := make(chan ExecutionChunk, 10)

	go func() {
		defer close(ch)

		for i, step := range plan {
			chunk := ExecutionChunk{
				Step:      step,
				Content:   fmt.Sprintf("Executing step %d: %s", i+1, step.Action),
				Timestamp: time.Now(),
			}

			select {
			case <-ctx.Done():
				ch <- ExecutionChunk{Err: ctx.Err()}
				return
			case ch <- chunk:
			}

			if e.streamingDelay > 0 && i < len(plan)-1 {
				select {
				case <-ctx.Done():
					ch <- ExecutionChunk{Err: ctx.Err()}
					return
				case <-time.After(e.streamingDelay):
				}
			}
		}

		// Send final chunk with answer
		select {
		case <-ctx.Done():
			ch <- ExecutionChunk{Err: ctx.Err()}
			return
		case ch <- ExecutionChunk{
			FinalAnswer: &schema.FinalAnswer{
				Output: fmt.Sprintf("Completed plan with %d steps", len(plan)),
			},
			Timestamp: time.Now(),
		}:
		}
	}()

	return ch, nil
}

// ConcurrentStreamingTestRunner runs concurrent streaming operations for testing.
type ConcurrentStreamingTestRunner struct {
	testFunc      func() error
	NumGoroutines int
	TestDuration  time.Duration
}

// NewConcurrentStreamingTestRunner creates a new concurrent streaming test runner.
func NewConcurrentStreamingTestRunner(numGoroutines int, duration time.Duration, testFunc func() error) *ConcurrentStreamingTestRunner {
	return &ConcurrentStreamingTestRunner{
		NumGoroutines: numGoroutines,
		TestDuration:  duration,
		testFunc:      testFunc,
	}
}

// Run executes the concurrent streaming test.
func (r *ConcurrentStreamingTestRunner) Run() error {
	var wg sync.WaitGroup
	errChan := make(chan error, r.NumGoroutines)
	stopChan := make(chan struct{})
	var stopOnce sync.Once

	stopFunc := func() {
		stopOnce.Do(func() {
			close(stopChan)
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.TestDuration+time.Second)
	defer cancel()

	timer := time.AfterFunc(r.TestDuration, stopFunc)
	defer func() {
		timer.Stop()
		stopFunc()
	}()

	for i := 0; i < r.NumGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				case <-ctx.Done():
					return
				default:
					if err := r.testFunc(); err != nil {
						select {
						case errChan <- err:
						default:
						}
						stopFunc()
						return
					}
				}
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		stopFunc()
		wg.Wait()
	}

	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// Streaming test helpers

// GenerateStreamChunks creates a slice of AgentStreamChunk from a string response.
func GenerateStreamChunks(response string, chunkSize int) []iface.AgentStreamChunk {
	if chunkSize <= 0 {
		chunkSize = 5 // Default chunk size
	}

	words := strings.Fields(response)
	chunks := make([]iface.AgentStreamChunk, 0)

	for i := 0; i < len(words); i += chunkSize {
		end := i + chunkSize
		if end > len(words) {
			end = len(words)
		}

		content := strings.Join(words[i:end], " ") + " "
		chunks = append(chunks, iface.AgentStreamChunk{
			Content: content,
			Metadata: map[string]any{
				"chunk_index": len(chunks),
				"timestamp":   time.Now(),
			},
		})
	}

	// Add final chunk with finish
	if len(chunks) > 0 {
		lastChunk := chunks[len(chunks)-1]
		chunks[len(chunks)-1] = iface.AgentStreamChunk{
			Content: lastChunk.Content,
			Finish: &iface.AgentFinish{
				ReturnValues: map[string]any{"output": response},
				Log:          "Streaming completed",
			},
			Metadata: lastChunk.Metadata,
		}
	}

	return chunks
}

// ValidateStreamChunk validates that a stream chunk has valid structure.
func ValidateStreamChunk(t *testing.T, chunk iface.AgentStreamChunk, allowEmpty bool) {
	t.Helper()
	if !allowEmpty {
		assert.True(t, chunk.Content != "" || chunk.ToolCalls != nil || chunk.Action != nil || chunk.Finish != nil || chunk.Err != nil,
			"Stream chunk should have at least one field set")
	}
	if chunk.Finish != nil && chunk.Err != nil {
		t.Errorf("Stream chunk should not have both Finish and Err set")
	}
}

// CollectStreamChunks collects all chunks from a stream channel.
func CollectStreamChunks(ch <-chan iface.AgentStreamChunk) ([]iface.AgentStreamChunk, error) {
	chunks := make([]iface.AgentStreamChunk, 0)
	for chunk := range ch {
		chunks = append(chunks, chunk)
		if chunk.Err != nil {
			return chunks, chunk.Err
		}
	}
	return chunks, nil
}
