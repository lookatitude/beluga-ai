// Package llms provides advanced test utilities and comprehensive mocks for testing LLM implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package llms

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockChatModel provides a comprehensive mock implementation for testing
type AdvancedMockChatModel struct {
	mock.Mock

	// Configuration
	modelName    string
	providerName string
	callCount    int
	mu           sync.RWMutex

	// Configurable behavior
	shouldError          bool
	errorToReturn        error
	responses            []string
	responseIndex        int
	streamingDelay       time.Duration
	simulateNetworkDelay bool

	// Tool support
	boundTools    []tools.Tool
	toolCallCount int
	toolResults   map[string]interface{}

	// Health check data
	healthState     string
	lastHealthCheck time.Time
}

// NewAdvancedMockChatModel creates a new advanced mock with configurable behavior
func NewAdvancedMockChatModel(modelName string, opts ...MockOption) *AdvancedMockChatModel {
	m := &AdvancedMockChatModel{
		modelName:       modelName,
		providerName:    "advanced-mock",
		responses:       []string{"Default mock response"},
		toolResults:     make(map[string]interface{}),
		healthState:     "healthy",
		lastHealthCheck: time.Now(),
		streamingDelay:  10 * time.Millisecond,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// MockOption configures the behavior of AdvancedMockChatModel
type MockOption func(*AdvancedMockChatModel)

// WithProviderName sets the provider name
func WithProviderName(name string) MockOption {
	return func(m *AdvancedMockChatModel) {
		m.providerName = name
	}
}

// WithResponses sets the responses to return
func WithResponses(responses ...string) MockOption {
	return func(m *AdvancedMockChatModel) {
		m.responses = responses
	}
}

// WithError configures the mock to return an error
func WithError(err error) MockOption {
	return func(m *AdvancedMockChatModel) {
		m.shouldError = true
		m.errorToReturn = err
	}
}

// WithStreamingDelay sets the delay between streaming chunks
func WithStreamingDelay(delay time.Duration) MockOption {
	return func(m *AdvancedMockChatModel) {
		m.streamingDelay = delay
	}
}

// WithNetworkDelay enables network delay simulation
func WithNetworkDelay(enabled bool) MockOption {
	return func(m *AdvancedMockChatModel) {
		m.simulateNetworkDelay = enabled
	}
}

// WithHealthState sets the health check state
func WithHealthState(state string) MockOption {
	return func(m *AdvancedMockChatModel) {
		m.healthState = state
	}
}

// WithToolResults pre-configures tool execution results
func WithToolResults(results map[string]interface{}) MockOption {
	return func(m *AdvancedMockChatModel) {
		for k, v := range results {
			m.toolResults[k] = v
		}
	}
}

// Generate implements the ChatModel interface
func (m *AdvancedMockChatModel) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		args := m.Called(ctx, messages, options)
		if args.Get(0) != nil {
			return args.Get(0).(schema.Message), args.Error(1)
		}
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, fmt.Errorf("mock error")
	}

	// Default behavior
	response := m.getNextResponse()
	return schema.NewAIMessage(response), nil
}

// StreamChat implements the ChatModel interface with realistic streaming
func (m *AdvancedMockChatModel) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		args := m.Called(ctx, messages, options)
		if args.Get(0) != nil {
			return args.Get(0).(<-chan iface.AIMessageChunk), args.Error(1)
		}
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, fmt.Errorf("mock error")
	}

	streamChan := make(chan iface.AIMessageChunk, 10)

	go func() {
		defer close(streamChan)

		response := m.getNextResponse()
		words := strings.Fields(response)

		for _, word := range words {
			if m.simulateNetworkDelay {
				select {
				case <-ctx.Done():
					streamChan <- iface.AIMessageChunk{Err: ctx.Err()}
					return
				case <-time.After(m.streamingDelay):
				}
			}

			chunk := iface.AIMessageChunk{
				Content: word + " ",
			}

			select {
			case streamChan <- chunk:
			case <-ctx.Done():
				streamChan <- iface.AIMessageChunk{Err: ctx.Err()}
				return
			}
		}

		// Send final chunk with tool calls if tools are bound
		if len(m.boundTools) > 0 && m.toolCallCount < 2 { // Simulate occasional tool calls
			m.toolCallCount++
			chunk := iface.AIMessageChunk{
				ToolCallChunks: []schema.ToolCallChunk{
					{
						Name:      "calculator",
						Arguments: `{"expression": "2 + 2"}`,
					},
				},
			}
			select {
			case streamChan <- chunk:
			case <-ctx.Done():
				streamChan <- iface.AIMessageChunk{Err: ctx.Err()}
			}
		}
	}()

	return streamChan, nil
}

// BindTools implements the ChatModel interface
func (m *AdvancedMockChatModel) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		args := m.Called(toolsToBind)
		if args.Get(0) != nil {
			return args.Get(0).(iface.ChatModel)
		}
	}

	m.mu.Lock()
	m.boundTools = make([]tools.Tool, len(toolsToBind))
	copy(m.boundTools, toolsToBind)
	m.mu.Unlock()

	return m
}

// GetModelName implements the ChatModel interface
func (m *AdvancedMockChatModel) GetModelName() string {
	return m.modelName
}

// GetProviderName returns the provider name
func (m *AdvancedMockChatModel) GetProviderName() string {
	return m.providerName
}

// Invoke implements the Runnable interface
func (m *AdvancedMockChatModel) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		args := m.Called(ctx, input, options)
		if args.Get(0) != nil {
			return args.Get(0), args.Error(1)
		}
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, fmt.Errorf("mock error")
	}

	messages, err := EnsureMessages(input)
	if err != nil {
		return nil, err
	}

	result, err := m.Generate(ctx, messages, options...)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Batch implements the Runnable interface
func (m *AdvancedMockChatModel) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		args := m.Called(ctx, inputs, options)
		if args.Get(0) != nil {
			return args.Get(0).([]any), args.Error(1)
		}
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, fmt.Errorf("mock error")
	}

	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := m.Invoke(ctx, input, options...)
		if err != nil {
			return nil, fmt.Errorf("batch item %d failed: %w", i, err)
		}
		results[i] = result
	}

	return results, nil
}

// Stream implements the Runnable interface
func (m *AdvancedMockChatModel) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		args := m.Called(ctx, input, options)
		if args.Get(0) != nil {
			return args.Get(0).(<-chan any), args.Error(1)
		}
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, fmt.Errorf("mock error")
	}

	messages, err := EnsureMessages(input)
	if err != nil {
		return nil, err
	}

	chunkChan, err := m.StreamChat(ctx, messages, options...)
	if err != nil {
		return nil, err
	}

	outputChan := make(chan any)
	go func() {
		defer close(outputChan)
		for chunk := range chunkChan {
			select {
			case outputChan <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	return outputChan, nil
}

// CheckHealth implements the HealthChecker interface
func (m *AdvancedMockChatModel) CheckHealth() map[string]interface{} {
	m.mu.Lock()
	m.lastHealthCheck = time.Now()
	health := map[string]interface{}{
		"state":           m.healthState,
		"provider":        m.providerName,
		"model":           m.modelName,
		"call_count":      m.callCount,
		"tools_bound":     len(m.boundTools),
		"timestamp":       m.lastHealthCheck.Unix(),
		"should_error":    m.shouldError,
		"responses_count": len(m.responses),
		"streaming_delay": m.streamingDelay.String(),
	}
	m.mu.Unlock()
	return health
}

// Helper methods

func (m *AdvancedMockChatModel) getNextResponse() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.responses) == 0 {
		return "Default mock response"
	}

	response := m.responses[m.responseIndex%len(m.responses)]
	m.responseIndex++
	return response
}

// GetCallCount returns the number of times methods were called
func (m *AdvancedMockChatModel) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// Reset resets the mock state
func (m *AdvancedMockChatModel) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
	m.responseIndex = 0
	m.toolCallCount = 0
	m.shouldError = false
	m.errorToReturn = nil
}

// AdvancedMockLLM provides a mock LLM implementation
type AdvancedMockLLM struct {
	mock.Mock
	modelName    string
	providerName string
	callCount    int
	mu           sync.RWMutex
}

func NewAdvancedMockLLM(modelName string) *AdvancedMockLLM {
	return &AdvancedMockLLM{
		modelName:    modelName,
		providerName: "advanced-mock-llm",
	}
}

func (m *AdvancedMockLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		args := m.Called(ctx, input, options)
		if args.Get(0) != nil {
			return args.Get(0), args.Error(1)
		}
	}
	return "Mock LLM response", nil
}

func (m *AdvancedMockLLM) GetModelName() string {
	return m.modelName
}

func (m *AdvancedMockLLM) GetProviderName() string {
	return m.providerName
}

// MockMetricsRecorder provides a mock implementation of MetricsRecorder
type MockMetricsRecorder struct {
	mock.Mock
}

func NewMockMetricsRecorder() *MockMetricsRecorder {
	return &MockMetricsRecorder{}
}

func (m *MockMetricsRecorder) RecordRequest(ctx context.Context, provider, model string, duration time.Duration) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		m.Called(ctx, provider, model, duration)
	}
}

func (m *MockMetricsRecorder) RecordError(ctx context.Context, provider, model, errorCode string, duration time.Duration) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		m.Called(ctx, provider, model, errorCode, duration)
	}
}

func (m *MockMetricsRecorder) RecordTokenUsage(ctx context.Context, provider, model string, inputTokens, outputTokens int) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		m.Called(ctx, provider, model, inputTokens, outputTokens)
	}
}

func (m *MockMetricsRecorder) RecordToolCall(ctx context.Context, provider, model, toolName string) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		m.Called(ctx, provider, model, toolName)
	}
}

func (m *MockMetricsRecorder) RecordBatch(ctx context.Context, provider, model string, batchSize int, duration time.Duration) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		m.Called(ctx, provider, model, batchSize, duration)
	}
}

func (m *MockMetricsRecorder) RecordStream(ctx context.Context, provider, model string, duration time.Duration) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		m.Called(ctx, provider, model, duration)
	}
}

func (m *MockMetricsRecorder) IncrementActiveRequests(ctx context.Context, provider, model string) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		m.Called(ctx, provider, model)
	}
}

func (m *MockMetricsRecorder) DecrementActiveRequests(ctx context.Context, provider, model string) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		m.Called(ctx, provider, model)
	}
}

func (m *MockMetricsRecorder) IncrementActiveStreams(ctx context.Context, provider, model string) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		m.Called(ctx, provider, model)
	}
}

func (m *MockMetricsRecorder) DecrementActiveStreams(ctx context.Context, provider, model string) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		m.Called(ctx, provider, model)
	}
}

// MockTracingHelper provides a mock implementation of tracing functionality
type MockTracingHelper struct {
	mock.Mock
}

func NewMockTracingHelper() *MockTracingHelper {
	return &MockTracingHelper{}
}

func (m *MockTracingHelper) StartOperation(ctx context.Context, operation string, provider, model string) context.Context {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		args := m.Called(ctx, operation, provider, model)
		if args.Get(0) != nil {
			return args.Get(0).(context.Context)
		}
	}
	return ctx
}

func (m *MockTracingHelper) RecordError(ctx context.Context, err error) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		m.Called(ctx, err)
	}
}

func (m *MockTracingHelper) AddSpanAttributes(ctx context.Context, attrs map[string]interface{}) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		m.Called(ctx, attrs)
	}
}

func (m *MockTracingHelper) EndSpan(ctx context.Context) {
	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		m.Called(ctx)
	}
}

// Test Utilities

// TestContext creates a context with timeout suitable for testing
func TestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// CreateTestMessages creates a set of test messages
func CreateTestMessages() []schema.Message {
	return []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant."),
		schema.NewHumanMessage("What is the capital of France?"),
		schema.NewAIMessage("The capital of France is Paris."),
		schema.NewHumanMessage("What is its population?"),
	}
}

// CreateTestConfig creates a test configuration
func CreateTestConfig() *Config {
	return NewConfig(
		WithProvider("mock"),
		WithModelName("test-model"),
		WithAPIKey("test-key"),
		WithTemperatureConfig(0.7),
		WithMaxTokensConfig(100),
		WithTimeout(10*time.Second),
	)
}

// AssertHealthCheck validates health check results
func AssertHealthCheck(t *testing.T, health map[string]interface{}) {
	assert.NotNil(t, health)
	assert.Contains(t, health, "state")
	assert.Contains(t, health, "provider")
	assert.Contains(t, health, "model")
	assert.Contains(t, health, "timestamp")
}

// AssertStreamingResponse validates streaming responses
func AssertStreamingResponse(t *testing.T, chunks <-chan iface.AIMessageChunk) {
	var receivedContent strings.Builder
	var chunkCount int

	for chunk := range chunks {
		chunkCount++
		assert.NoError(t, chunk.Err)
		receivedContent.WriteString(chunk.Content)
	}

	assert.Greater(t, chunkCount, 0, "Should receive at least one chunk")
	assert.NotEmpty(t, receivedContent.String(), "Should receive content")
}

// AssertErrorType checks if an error matches expected type
func AssertErrorType(t *testing.T, err error, expectedCode string) {
	if err == nil {
		t.Errorf("Expected error with code %s, but got nil", expectedCode)
		return
	}

	if llmErr := GetLLMError(err); llmErr != nil {
		assert.Equal(t, expectedCode, llmErr.Code)
	} else {
		t.Errorf("Expected LLMError with code %s, but got %T: %v", expectedCode, err, err)
	}
}

// ConcurrentTestRunner runs a test function concurrently
func ConcurrentTestRunner(t *testing.T, testFunc func(t *testing.T), goroutines int) {
	var wg sync.WaitGroup
	errChan := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("goroutine %d panicked: %v", id, r)
				}
			}()

			testFunc(t)
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Error(err)
	}
}

// LoadTestScenario represents a load testing scenario
type LoadTestScenario struct {
	Name        string
	Duration    time.Duration
	Concurrency int
	RequestRate int // requests per second, 0 for unlimited
	TestFunc    func(ctx context.Context) error
}

// RunLoadTest executes a load test scenario
func RunLoadTest(t *testing.T, scenario LoadTestScenario) {
	ctx, cancel := context.WithTimeout(context.Background(), scenario.Duration)
	defer cancel()

	var wg sync.WaitGroup
	results := make(chan error, scenario.Concurrency*10)

	start := time.Now()

	for i := 0; i < scenario.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					if err := scenario.TestFunc(ctx); err != nil {
						results <- err
					}

					if scenario.RequestRate > 0 {
						time.Sleep(time.Second / time.Duration(scenario.RequestRate))
					}
				}
			}
		}()
	}

	wg.Wait()
	close(results)

	elapsed := time.Since(start)
	var errors []error
	for err := range results {
		errors = append(errors, err)
	}

	t.Logf("Load test %s completed in %v", scenario.Name, elapsed)
	t.Logf("Total errors: %d", len(errors))

	if len(errors) > 0 {
		sampleSize := 3
		if len(errors) < 3 {
			sampleSize = len(errors)
		}
		t.Logf("Sample errors: %v", errors[:sampleSize])
	}
}

// MockTool provides a mock tool implementation for testing
type MockTool struct {
	name        string
	description string
	shouldError bool
	result      interface{}
}

func NewMockTool(name string) *MockTool {
	return &MockTool{
		name:        name,
		description: "Mock tool for testing",
	}
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) Definition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Name:        m.name,
		Description: m.description,
		InputSchema: "{}",
	}
}

func (m *MockTool) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	if m.shouldError {
		return nil, errors.New("mock tool error")
	}
	if m.result != nil {
		return m.result, nil
	}
	return fmt.Sprintf("Mock result from %s", m.name), nil
}

func (m *MockTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
	results := make([]any, len(inputs))
	for i := range inputs {
		result, err := m.Execute(ctx, inputs[i])
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (m *MockTool) SetShouldError(shouldError bool) {
	m.shouldError = shouldError
}

func (m *MockTool) SetResult(result interface{}) {
	m.result = result
}

// TestProviderInterface tests that a provider implements the ChatModel interface correctly
func TestProviderInterface(t *testing.T, provider iface.ChatModel, providerName string) {
	// Test basic properties
	assert.NotEmpty(t, provider.GetModelName(), "Provider should have a model name")
	assert.NotEmpty(t, provider.GetProviderName(), "Provider should have a provider name")

	// Test health check
	health := provider.CheckHealth()
	AssertHealthCheck(t, health)

	// Test message generation
	ctx := context.Background()
	messages := CreateTestMessages()

	response, err := provider.Generate(ctx, messages)
	assert.NoError(t, err, "Generate should not error")
	assert.NotNil(t, response, "Generate should return a response")
	assert.NotEmpty(t, response.GetContent(), "Response should have content")

	// Test streaming
	streamChan, err := provider.StreamChat(ctx, messages)
	assert.NoError(t, err, "StreamChat should not error")
	AssertStreamingResponse(t, streamChan)

	// Test tool binding
	tools := []tools.Tool{
		NewMockTool("test-tool"),
	}
	boundProvider := provider.BindTools(tools)
	assert.NotNil(t, boundProvider, "BindTools should return a provider")

	// Test Runnable interface
	result, err := provider.Invoke(ctx, "test input")
	assert.NoError(t, err, "Invoke should not error")
	assert.NotNil(t, result, "Invoke should return a result")

	// Test batch processing
	inputs := []any{"input1", "input2", "input3"}
	results, err := provider.Batch(ctx, inputs)
	assert.NoError(t, err, "Batch should not error")
	assert.Len(t, results, len(inputs), "Batch should return correct number of results")

	// Test streaming interface
	streamResult, err := provider.Stream(ctx, "test input")
	assert.NoError(t, err, "Stream should not error")

	// Collect streaming results
	var streamResults []any
	for result := range streamResult {
		streamResults = append(streamResults, result)
	}
	assert.NotEmpty(t, streamResults, "Stream should return results")
}

// Note: IntegrationTestHelper is defined in integration_test_setup.go
// This file uses the IntegrationTestHelper from that file
