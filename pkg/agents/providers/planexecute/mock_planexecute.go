// Package planexecute provides mock implementations and test utilities for Plan-and-Execute agents.
package planexecute

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// MockPlanExecuteAgent provides a mock implementation of PlanExecuteAgent for testing.
type MockPlanExecuteAgent struct {
	errorToReturn error
	*PlanExecuteAgent
	planSteps        []PlanStep
	executionResults []map[string]any
	callCount        int
	simulateDelay    time.Duration
	mu               sync.RWMutex
	shouldError      bool
}

// MockPlanExecuteOption defines functional options for configuring the mock.
type MockPlanExecuteOption func(*MockPlanExecuteAgent)

// WithMockPlanExecuteError configures the mock to return errors.
func WithMockPlanExecuteError(shouldError bool, err error) MockPlanExecuteOption {
	return func(m *MockPlanExecuteAgent) {
		m.shouldError = shouldError
		m.errorToReturn = err
	}
}

// WithMockPlanSteps sets predefined plan steps for the mock.
func WithMockPlanSteps(steps []PlanStep) MockPlanExecuteOption {
	return func(m *MockPlanExecuteAgent) {
		m.planSteps = steps
	}
}

// WithMockExecutionResults sets predefined execution results.
func WithMockExecutionResults(results []map[string]any) MockPlanExecuteOption {
	return func(m *MockPlanExecuteAgent) {
		m.executionResults = results
	}
}

// WithMockPlanExecuteDelay adds artificial delay to mock operations.
func WithMockPlanExecuteDelay(delay time.Duration) MockPlanExecuteOption {
	return func(m *MockPlanExecuteAgent) {
		m.simulateDelay = delay
	}
}

// NewMockPlanExecuteAgent creates a new mock PlanExecuteAgent.
// This is a convenience function for testing that creates a mock with a mock ChatModel.
func NewMockPlanExecuteAgent(name string, mockLLM llmsiface.ChatModel, agentTools []iface.Tool, opts ...MockPlanExecuteOption) (*MockPlanExecuteAgent, error) {
	agent, err := NewPlanExecuteAgent(name, mockLLM, agentTools)
	if err != nil {
		return nil, err
	}

	mock := &MockPlanExecuteAgent{
		PlanExecuteAgent: agent,
		planSteps:        []PlanStep{},
		executionResults: []map[string]any{},
	}

	for _, opt := range opts {
		opt(mock)
	}

	return mock, nil
}

// GetCallCount returns the number of times the mock has been called.
func (m *MockPlanExecuteAgent) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetPlanSteps returns the plan steps that were set for the mock.
func (m *MockPlanExecuteAgent) GetPlanSteps() []PlanStep {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]PlanStep, len(m.planSteps))
	copy(result, m.planSteps)
	return result
}

// GetExecutionResults returns the execution results that were set for the mock.
func (m *MockPlanExecuteAgent) GetExecutionResults() []map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]map[string]any, len(m.executionResults))
	copy(result, m.executionResults)
	return result
}

// Plan overrides the embedded PlanExecuteAgent.Plan method to provide mock behavior.
// This method increments call count, simulates delays, and returns configured errors.
func (m *MockPlanExecuteAgent) Plan(ctx context.Context, intermediateSteps []iface.IntermediateStep, inputs map[string]any) (iface.AgentAction, iface.AgentFinish, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	// Simulate delay if configured
	if m.simulateDelay > 0 {
		select {
		case <-time.After(m.simulateDelay):
		case <-ctx.Done():
			return iface.AgentAction{}, iface.AgentFinish{}, ctx.Err()
		}
	}

	// Return error if configured
	if m.shouldError {
		return iface.AgentAction{}, iface.AgentFinish{}, m.errorToReturn
	}

	// If we have predefined plan steps, create an execution plan and return action to execute it
	if len(m.planSteps) > 0 {
		plan := &ExecutionPlan{
			Goal:       "mock goal",
			Steps:      m.planSteps,
			TotalSteps: len(m.planSteps),
		}

		// Return an action to execute the plan
		action := iface.AgentAction{
			Tool:      "ExecutePlan",
			ToolInput: map[string]any{"plan": plan},
			Log:       "Mock planning completed",
		}
		return action, iface.AgentFinish{}, nil
	}

	// Default behavior: return a finish with mock results (no planning needed)
	finish := iface.AgentFinish{
		ReturnValues: map[string]any{
			"output": "Mock plan execution completed",
		},
		Log: "Mock planning and execution completed",
	}
	return iface.AgentAction{}, finish, nil
}

// ExecutePlan overrides the embedded PlanExecuteAgent.ExecutePlan method to provide mock behavior.
// This method increments call count, simulates delays, returns configured errors, and uses mock execution results.
func (m *MockPlanExecuteAgent) ExecutePlan(ctx context.Context, plan *ExecutionPlan) (map[string]any, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	// Simulate delay if configured
	if m.simulateDelay > 0 {
		select {
		case <-time.After(m.simulateDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Return error if configured
	if m.shouldError {
		return nil, m.errorToReturn
	}

	// If we have predefined execution results, return them
	if len(m.executionResults) > 0 {
		results := make(map[string]any)
		for i, result := range m.executionResults {
			results[fmt.Sprintf("step_%d", i+1)] = result
		}
		results["total_steps"] = len(m.executionResults)
		return results, nil
	}

	// Default mock execution results
	results := make(map[string]any)
	for _, step := range plan.Steps {
		results[fmt.Sprintf("step_%d", step.StepNumber)] = map[string]any{
			"tool":        step.Tool,
			"input":       step.Input,
			"observation": fmt.Sprintf("Mock execution result for step %d", step.StepNumber),
		}
	}
	results["total_steps"] = len(plan.Steps)
	return results, nil
}

// CreateTestPlanExecuteAgent creates a test PlanExecuteAgent with default configuration.
func CreateTestPlanExecuteAgent(name string, mockLLM llmsiface.ChatModel, tools []iface.Tool) (*PlanExecuteAgent, error) {
	return NewPlanExecuteAgent(name, mockLLM, tools)
}

// CreateTestPlanSteps creates a slice of test plan steps.
func CreateTestPlanSteps(count int) []PlanStep {
	steps := make([]PlanStep, count)
	for i := 0; i < count; i++ {
		steps[i] = PlanStep{
			StepNumber: i + 1,
			Action:     "test_action",
			Tool:       "test_tool",
			Input:      "test_input",
			Reasoning:  "test_reasoning",
		}
	}
	return steps
}

// CreateTestExecutionPlan creates a test execution plan.
func CreateTestExecutionPlan(goal string, stepCount int) *ExecutionPlan {
	return &ExecutionPlan{
		Goal:       goal,
		Steps:      CreateTestPlanSteps(stepCount),
		TotalSteps: stepCount,
	}
}

// MockChatModelForPlanExecute provides a minimal mock ChatModel for PlanExecute testing.
type MockChatModelForPlanExecute struct {
	errorToReturn error
	responses     []schema.Message
	responseIndex int
	mu            sync.RWMutex
	shouldError   bool
}

// NewMockChatModelForPlanExecute creates a new mock ChatModel for PlanExecute testing.
func NewMockChatModelForPlanExecute() *MockChatModelForPlanExecute {
	return &MockChatModelForPlanExecute{
		responses: []schema.Message{},
	}
}

// WithMockResponse sets a response to return.
func (m *MockChatModelForPlanExecute) WithMockResponse(msg schema.Message) *MockChatModelForPlanExecute {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = append(m.responses, msg)
	return m
}

// WithMockError configures the mock to return an error.
func (m *MockChatModelForPlanExecute) WithMockError(err error) *MockChatModelForPlanExecute {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = true
	m.errorToReturn = err
	return m
}

// Generate implements llmsiface.ChatModel interface.
func (m *MockChatModelForPlanExecute) Generate(ctx context.Context, messages []schema.Message, options ...any) (schema.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, errors.New("mock error")
	}

	if len(m.responses) > m.responseIndex {
		response := m.responses[m.responseIndex]
		m.responseIndex = (m.responseIndex + 1) % len(m.responses)
		return response, nil
	}

	return schema.NewAIMessage("Mock response"), nil
}

// Note: This is a minimal mock. Full ChatModel interface implementation would require
// implementing all methods from llmsiface.ChatModel. For testing purposes, this provides
// the basic Generate method needed for PlanExecute agent testing.
