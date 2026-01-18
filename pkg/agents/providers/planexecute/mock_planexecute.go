// Package planexecute provides mock implementations and test utilities for Plan-and-Execute agents.
package planexecute

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// MockPlanExecuteAgent provides a mock implementation of PlanExecuteAgent for testing.
type MockPlanExecuteAgent struct {
	*PlanExecuteAgent
	shouldError      bool
	errorToReturn    error
	planSteps        []PlanStep
	executionResults []map[string]any
	callCount        int
	mu               sync.RWMutex
	simulateDelay    time.Duration
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
func NewMockPlanExecuteAgent(name string, mockLLM llmsiface.ChatModel, agentTools []tools.Tool, opts ...MockPlanExecuteOption) (*MockPlanExecuteAgent, error) {
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

// CreateTestPlanExecuteAgent creates a test PlanExecuteAgent with default configuration.
func CreateTestPlanExecuteAgent(name string, mockLLM llmsiface.ChatModel, tools []tools.Tool) (*PlanExecuteAgent, error) {
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
			Input:       "test_input",
			Reasoning:   "test_reasoning",
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
	shouldError   bool
	errorToReturn error
	responses     []schema.Message
	responseIndex int
	mu            sync.RWMutex
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
func (m *MockChatModelForPlanExecute) Generate(ctx context.Context, messages []schema.Message, options ...interface{}) (schema.Message, error) {
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
