// Package chatmodels provides advanced test utilities and comprehensive mocks for testing chat model implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package chatmodels

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
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
	shouldError      bool
	errorToReturn    error
	responses        []schema.Message
	responseIndex    int
	streamingDelay   time.Duration
	simulateFailures bool

	// Chat model specific
	conversationHistory []schema.Message
	toolsSupported      bool

	// Health check data
	healthState     string
	lastHealthCheck time.Time
}

// NewAdvancedMockChatModel creates a new advanced mock with configurable behavior
func NewAdvancedMockChatModel(modelName, providerName string, options ...MockChatModelOption) *AdvancedMockChatModel {
	mock := &AdvancedMockChatModel{
		modelName:           modelName,
		providerName:        providerName,
		responses:           []schema.Message{},
		conversationHistory: []schema.Message{},
		toolsSupported:      true,
		healthState:         "healthy",
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	// Set default responses if none provided
	if len(mock.responses) == 0 {
		mock.responses = []schema.Message{
			schema.NewAIMessage("This is a default mock response from the chat model."),
			schema.NewAIMessage("Another default response for conversation flow testing."),
		}
	}

	return mock
}

// MockChatModelOption defines functional options for mock configuration
type MockChatModelOption func(*AdvancedMockChatModel)

// WithMockError configures the mock to return errors
func WithMockError(shouldError bool, err error) MockChatModelOption {
	return func(m *AdvancedMockChatModel) {
		m.shouldError = shouldError
		m.errorToReturn = err
	}
}

// WithMockResponses sets predefined responses for the mock
func WithMockResponses(responses []schema.Message) MockChatModelOption {
	return func(m *AdvancedMockChatModel) {
		m.responses = make([]schema.Message, len(responses))
		copy(m.responses, responses)
	}
}

// WithStreamingDelay adds artificial delay to mock operations
func WithStreamingDelay(delay time.Duration) MockChatModelOption {
	return func(m *AdvancedMockChatModel) {
		m.streamingDelay = delay
	}
}

// WithToolsSupport configures whether the mock supports tools
func WithToolsSupport(supported bool) MockChatModelOption {
	return func(m *AdvancedMockChatModel) {
		m.toolsSupported = supported
	}
}

// WithConversationHistory preloads conversation history
func WithConversationHistory(messages []schema.Message) MockChatModelOption {
	return func(m *AdvancedMockChatModel) {
		m.conversationHistory = make([]schema.Message, len(messages))
		copy(m.conversationHistory, messages)
	}
}

// Mock implementation methods for core.Runnable interface
func (m *AdvancedMockChatModel) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	// For chat models, input is typically []schema.Message
	if messages, ok := input.([]schema.Message); ok {
		response, err := m.Generate(ctx, messages, options...)
		return response, err
	}

	// Handle string input by converting to message
	if str, ok := input.(string); ok {
		messages := []schema.Message{schema.NewHumanMessage(str)}
		response, err := m.Generate(ctx, messages, options...)
		return response, err
	}

	return nil, fmt.Errorf("unsupported input type for chat model: %T", input)
}

func (m *AdvancedMockChatModel) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := m.Invoke(ctx, input, options...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (m *AdvancedMockChatModel) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	// For chat models, streaming returns AIMessageChunk
	if messages, ok := input.([]schema.Message); ok {
		chunkCh, err := m.StreamChat(ctx, messages, options...)
		if err != nil {
			return nil, err
		}

		// Convert AIMessageChunk channel to any channel
		resultCh := make(chan any, 1)
		go func() {
			defer close(resultCh)
			for chunk := range chunkCh {
				resultCh <- chunk
			}
		}()

		return resultCh, nil
	}

	return nil, fmt.Errorf("streaming input must be []schema.Message, got %T", input)
}

// Mock implementation methods for ChatModel interface
func (m *AdvancedMockChatModel) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.streamingDelay > 0 {
		time.Sleep(m.streamingDelay)
	}

	if m.shouldError {
		return nil, m.errorToReturn
	}

	// Add messages to conversation history
	m.mu.Lock()
	m.conversationHistory = append(m.conversationHistory, messages...)
	m.mu.Unlock()

	// Return next response
	if len(m.responses) > m.responseIndex {
		response := m.responses[m.responseIndex]
		m.responseIndex = (m.responseIndex + 1) % len(m.responses)

		// Add response to conversation history
		m.mu.Lock()
		m.conversationHistory = append(m.conversationHistory, response)
		m.mu.Unlock()

		return response, nil
	}

	// Default response
	defaultResponse := schema.NewAIMessage(fmt.Sprintf("Mock response from %s for %d messages", m.modelName, len(messages)))
	m.mu.Lock()
	m.conversationHistory = append(m.conversationHistory, defaultResponse)
	m.mu.Unlock()

	return defaultResponse, nil
}

func (m *AdvancedMockChatModel) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.shouldError {
		return nil, m.errorToReturn
	}

	ch := make(chan llmsiface.AIMessageChunk, 5)

	go func() {
		defer close(ch)

		// Simulate streaming response
		responseText := fmt.Sprintf("Streaming response from %s", m.modelName)

		// Split response into chunks
		chunkSize := len(responseText) / 3
		if chunkSize == 0 {
			chunkSize = 1
		}

		for i := 0; i < len(responseText); i += chunkSize {
			if m.streamingDelay > 0 {
				time.Sleep(m.streamingDelay)
			}

			end := i + chunkSize
			if end > len(responseText) {
				end = len(responseText)
			}

			chunk := llmsiface.AIMessageChunk{
				Content:        responseText[i:end],
				ToolCallChunks: []schema.ToolCallChunk{},
				AdditionalArgs: make(map[string]interface{}),
				Err:            nil,
			}

			ch <- chunk
		}
	}()

	return ch, nil
}

func (m *AdvancedMockChatModel) BindTools(toolsToBind []tools.Tool) llmsiface.ChatModel {
	// Return a new instance with tools bound
	newMock := &AdvancedMockChatModel{
		modelName:           m.modelName,
		providerName:        m.providerName,
		responses:           make([]schema.Message, len(m.responses)),
		conversationHistory: make([]schema.Message, len(m.conversationHistory)),
		toolsSupported:      true,
		healthState:         m.healthState,
	}

	copy(newMock.responses, m.responses)
	copy(newMock.conversationHistory, m.conversationHistory)

	return newMock
}

func (m *AdvancedMockChatModel) GetModelName() string {
	return m.modelName
}

func (m *AdvancedMockChatModel) GetProviderName() string {
	return m.providerName
}

func (m *AdvancedMockChatModel) CheckHealth() map[string]interface{} {
	m.lastHealthCheck = time.Now()
	return map[string]interface{}{
		"status":              m.healthState,
		"model_name":          m.modelName,
		"provider_name":       m.providerName,
		"call_count":          m.callCount,
		"conversation_length": len(m.conversationHistory),
		"tools_supported":     m.toolsSupported,
		"last_checked":        m.lastHealthCheck,
	}
}

// Additional helper methods for testing
func (m *AdvancedMockChatModel) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

func (m *AdvancedMockChatModel) GetConversationHistory() []schema.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]schema.Message, len(m.conversationHistory))
	copy(result, m.conversationHistory)
	return result
}

func (m *AdvancedMockChatModel) ResetConversation() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conversationHistory = []schema.Message{}
	m.responseIndex = 0
}

// Test data creation helpers

// CreateTestMessages creates a set of test messages for chat model testing
func CreateTestMessages(conversationLength int) []schema.Message {
	messages := make([]schema.Message, 0, conversationLength*2)

	topics := []string{"AI", "ML", "DL", "NLP", "robotics"}

	for i := 0; i < conversationLength; i++ {
		topic := topics[i%len(topics)]

		humanMsg := schema.NewHumanMessage(fmt.Sprintf("Tell me about %s (turn %d)", topic, i+1))
		aiMsg := schema.NewAIMessage(fmt.Sprintf("Here's information about %s: It's a fascinating field... (response %d)", topic, i+1))

		messages = append(messages, humanMsg, aiMsg)
	}

	return messages
}

// CreateTestChatModelConfig creates a test chat model configuration
func CreateTestChatModelConfig() Config {
	return Config{
		DefaultProvider:    "mock",
		DefaultModel:       "mock-chat-model",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   1000,
		DefaultTopP:        0.9,
		DefaultTimeout:     30 * time.Second,
		EnableMetrics:      true,
		EnableTracing:      true,
		MetricsPrefix:      "test_chatmodels",
		TracingServiceName: "test-chatmodels",
	}
}

// Assertion helpers

// AssertChatResponse validates chat model response
func AssertChatResponse(t *testing.T, response schema.Message, expectedMinLength int) {
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.GetContent())
	assert.GreaterOrEqual(t, len(response.GetContent()), expectedMinLength)
	assert.Equal(t, schema.RoleAssistant, response.GetType())
}

// AssertStreamingResponse validates streaming response
func AssertStreamingResponse(t *testing.T, chunks []llmsiface.AIMessageChunk, expectedMinChunks int) {
	assert.GreaterOrEqual(t, len(chunks), expectedMinChunks)

	fullContent := ""
	for i, chunk := range chunks {
		assert.NoError(t, chunk.Err, "Chunk %d should not have error", i)
		fullContent += chunk.Content
	}

	assert.NotEmpty(t, fullContent, "Combined streaming content should not be empty")
}

// AssertConversationFlow validates conversation flow
func AssertConversationFlow(t *testing.T, history []schema.Message, expectedMinLength int) {
	assert.GreaterOrEqual(t, len(history), expectedMinLength)

	// Verify alternating human/AI pattern
	for i, msg := range history {
		if i%2 == 0 {
			assert.Equal(t, schema.RoleHuman, msg.GetType(), "Message %d should be human", i)
		} else {
			assert.Equal(t, schema.RoleAssistant, msg.GetType(), "Message %d should be AI", i)
		}
		assert.NotEmpty(t, msg.GetContent(), "Message %d should have content", i)
	}
}

// AssertChatModelHealth validates chat model health check results
func AssertChatModelHealth(t *testing.T, health map[string]interface{}, expectedStatus string) {
	assert.Contains(t, health, "status")
	assert.Equal(t, expectedStatus, health["status"])
	assert.Contains(t, health, "model_name")
	assert.Contains(t, health, "provider_name")
	assert.Contains(t, health, "call_count")
}

// AssertErrorType validates error types and codes
func AssertErrorType(t *testing.T, err error, expectedCode string) {
	assert.Error(t, err)
	var chatErr *ChatModelError
	if assert.ErrorAs(t, err, &chatErr) {
		assert.Equal(t, expectedCode, chatErr.Code)
	}
}

// Performance testing helpers

// ConcurrentTestRunner runs chat model tests concurrently for performance testing
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

// RunLoadTest executes a load test scenario on chat model
func RunLoadTest(t *testing.T, chatModel *AdvancedMockChatModel, numOperations int, concurrency int) {
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
				// Test Generate
				messages := []schema.Message{schema.NewHumanMessage(fmt.Sprintf("Test message %d", opID))}
				_, err := chatModel.Generate(ctx, messages)
				if err != nil {
					errChan <- err
				}
			} else {
				// Test StreamChat
				messages := []schema.Message{schema.NewHumanMessage(fmt.Sprintf("Streaming test %d", opID))}
				streamCh, err := chatModel.StreamChat(ctx, messages)
				if err != nil {
					errChan <- err
					return
				}

				// Consume stream
				for chunk := range streamCh {
					if chunk.Err != nil {
						errChan <- chunk.Err
						return
					}
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
	assert.Equal(t, numOperations, chatModel.GetCallCount())
}

// Integration test helpers

// IntegrationTestHelper provides utilities for integration testing
type IntegrationTestHelper struct {
	chatModels map[string]*AdvancedMockChatModel
}

func NewIntegrationTestHelper() *IntegrationTestHelper {
	return &IntegrationTestHelper{
		chatModels: make(map[string]*AdvancedMockChatModel),
	}
}

func (h *IntegrationTestHelper) AddChatModel(name string, chatModel *AdvancedMockChatModel) {
	h.chatModels[name] = chatModel
}

func (h *IntegrationTestHelper) GetChatModel(name string) *AdvancedMockChatModel {
	return h.chatModels[name]
}

func (h *IntegrationTestHelper) Reset() {
	for _, chatModel := range h.chatModels {
		chatModel.ResetConversation()
		chatModel.callCount = 0
	}
}

// ChatModelScenarioRunner runs common chat model scenarios
type ChatModelScenarioRunner struct {
	chatModel llmsiface.ChatModel
}

func NewChatModelScenarioRunner(chatModel llmsiface.ChatModel) *ChatModelScenarioRunner {
	return &ChatModelScenarioRunner{
		chatModel: chatModel,
	}
}

func (r *ChatModelScenarioRunner) RunConversationScenario(ctx context.Context, turns int) error {
	messages := make([]schema.Message, 0, turns*2)

	for i := 0; i < turns; i++ {
		// Add human message
		humanMsg := schema.NewHumanMessage(fmt.Sprintf("This is turn %d in our conversation", i+1))
		messages = append(messages, humanMsg)

		// Generate AI response
		response, err := r.chatModel.Generate(ctx, messages)
		if err != nil {
			return fmt.Errorf("turn %d failed: %w", i+1, err)
		}

		// Add AI response to conversation
		messages = append(messages, response)
	}

	return nil
}

func (r *ChatModelScenarioRunner) RunStreamingScenario(ctx context.Context, queries []string) error {
	for i, query := range queries {
		messages := []schema.Message{schema.NewHumanMessage(query)}

		streamCh, err := r.chatModel.StreamChat(ctx, messages)
		if err != nil {
			return fmt.Errorf("streaming query %d failed: %w", i+1, err)
		}

		// Collect stream
		var fullResponse string
		chunkCount := 0
		for chunk := range streamCh {
			if chunk.Err != nil {
				return fmt.Errorf("streaming chunk error in query %d: %w", i+1, chunk.Err)
			}
			fullResponse += chunk.Content
			chunkCount++
		}

		if fullResponse == "" {
			return fmt.Errorf("streaming query %d produced empty response", i+1)
		}

		if chunkCount == 0 {
			return fmt.Errorf("streaming query %d produced no chunks", i+1)
		}
	}

	return nil
}

// BenchmarkHelper provides benchmarking utilities for chat models
type BenchmarkHelper struct {
	chatModel    llmsiface.ChatModel
	testMessages [][]schema.Message
}

func NewBenchmarkHelper(chatModel llmsiface.ChatModel, conversationCount int) *BenchmarkHelper {
	testMessages := make([][]schema.Message, conversationCount)
	for i := 0; i < conversationCount; i++ {
		testMessages[i] = CreateTestMessages(3) // 3-turn conversations
	}

	return &BenchmarkHelper{
		chatModel:    chatModel,
		testMessages: testMessages,
	}
}

func (b *BenchmarkHelper) BenchmarkGeneration(iterations int) (time.Duration, error) {
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		messages := b.testMessages[i%len(b.testMessages)]
		_, err := b.chatModel.Generate(ctx, messages)
		if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}

func (b *BenchmarkHelper) BenchmarkStreaming(iterations int) (time.Duration, error) {
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		messages := b.testMessages[i%len(b.testMessages)]
		streamCh, err := b.chatModel.StreamChat(ctx, messages)
		if err != nil {
			return 0, err
		}

		// Consume stream
		for chunk := range streamCh {
			if chunk.Err != nil {
				return 0, chunk.Err
			}
		}
	}

	return time.Since(start), nil
}
