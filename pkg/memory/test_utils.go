// Package memory provides advanced test utilities and comprehensive mocks for testing memory implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package memory

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockMemory provides a comprehensive mock implementation for testing
type AdvancedMockMemory struct {
	mock.Mock

	// Configuration
	memoryKey  string
	memoryType MemoryType
	callCount  int
	mu         sync.RWMutex

	// Configurable behavior
	shouldError     bool
	errorToReturn   error
	memoryVariables []string
	storedContext   map[string]interface{}
	returnMessages  bool
	simulateDelay   time.Duration

	// Memory-specific data
	messages       []schema.Message
	contextHistory []map[string]interface{}
	maxMemorySize  int

	// Health check data
	healthState     string
	lastHealthCheck time.Time
}

// NewAdvancedMockMemory creates a new advanced mock with configurable behavior
func NewAdvancedMockMemory(memoryKey string, memoryType MemoryType, options ...MockMemoryOption) *AdvancedMockMemory {
	mock := &AdvancedMockMemory{
		memoryKey:       memoryKey,
		memoryType:      memoryType,
		memoryVariables: []string{memoryKey},
		storedContext:   make(map[string]interface{}),
		messages:        make([]schema.Message, 0),
		contextHistory:  make([]map[string]interface{}, 0),
		maxMemorySize:   100,
		healthState:     "healthy",
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	return mock
}

// MockMemoryOption defines functional options for mock configuration
type MockMemoryOption func(*AdvancedMockMemory)

// WithMockError configures the mock to return errors
func WithMockError(shouldError bool, err error) MockMemoryOption {
	return func(m *AdvancedMockMemory) {
		m.shouldError = shouldError
		m.errorToReturn = err
	}
}

// WithMemoryVariables sets the memory variables for the mock
func WithMemoryVariables(variables []string) MockMemoryOption {
	return func(m *AdvancedMockMemory) {
		m.memoryVariables = variables
	}
}

// WithMockReturnMessages sets whether to return messages directly
func WithMockReturnMessages(returnMessages bool) MockMemoryOption {
	return func(m *AdvancedMockMemory) {
		m.returnMessages = returnMessages
	}
}

// WithSimulateDelay adds artificial delay to mock operations
func WithSimulateDelay(delay time.Duration) MockMemoryOption {
	return func(m *AdvancedMockMemory) {
		m.simulateDelay = delay
	}
}

// WithMaxMemorySize sets the maximum memory size
func WithMaxMemorySize(size int) MockMemoryOption {
	return func(m *AdvancedMockMemory) {
		m.maxMemorySize = size
	}
}

// WithPreloadedMessages preloads messages into the mock
func WithPreloadedMessages(messages []schema.Message) MockMemoryOption {
	return func(m *AdvancedMockMemory) {
		m.messages = make([]schema.Message, len(messages))
		copy(m.messages, messages)
	}
}

// Mock implementation methods
func (m *AdvancedMockMemory) MemoryVariables() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.memoryVariables
}

func (m *AdvancedMockMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		return nil, m.errorToReturn
	}

	result := make(map[string]any)
	if m.returnMessages {
		result[m.memoryKey] = m.messages
	} else {
		result[m.memoryKey] = GetBufferString(m.messages, "Human", "AI")
	}

	// Add stored context
	for k, v := range m.storedContext {
		result[k] = v
	}

	return result, nil
}

func (m *AdvancedMockMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		return m.errorToReturn
	}

	// Store the context
	context := make(map[string]interface{})
	for k, v := range inputs {
		context["input_"+k] = v
	}
	for k, v := range outputs {
		context["output_"+k] = v
	}

	m.contextHistory = append(m.contextHistory, context)

	// Extract and store messages if present
	if inputMsg, ok := inputs["input"]; ok {
		if msgStr, ok := inputMsg.(string); ok {
			humanMsg := schema.NewHumanMessage(msgStr)
			m.messages = append(m.messages, humanMsg)
		}
	}

	if outputMsg, ok := outputs["output"]; ok {
		if msgStr, ok := outputMsg.(string); ok {
			aiMsg := schema.NewAIMessage(msgStr)
			m.messages = append(m.messages, aiMsg)
		}
	}

	// Enforce memory size limits
	if len(m.messages) > m.maxMemorySize {
		m.messages = m.messages[len(m.messages)-m.maxMemorySize:]
	}

	return nil
}

func (m *AdvancedMockMemory) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		return m.errorToReturn
	}

	m.messages = make([]schema.Message, 0)
	m.contextHistory = make([]map[string]interface{}, 0)
	m.storedContext = make(map[string]interface{})

	return nil
}

func (m *AdvancedMockMemory) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

func (m *AdvancedMockMemory) GetMessages() []schema.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]schema.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

func (m *AdvancedMockMemory) GetContextHistory() []map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]map[string]interface{}, len(m.contextHistory))
	copy(result, m.contextHistory)
	return result
}

func (m *AdvancedMockMemory) CheckHealth() map[string]interface{} {
	m.lastHealthCheck = time.Now()
	return map[string]interface{}{
		"status":          m.healthState,
		"memory_key":      m.memoryKey,
		"memory_type":     string(m.memoryType),
		"call_count":      m.callCount,
		"message_count":   len(m.messages),
		"context_history": len(m.contextHistory),
		"last_checked":    m.lastHealthCheck,
	}
}

// AdvancedMockChatMessageHistory provides a comprehensive mock for chat message history
type AdvancedMockChatMessageHistory struct {
	mock.Mock

	// Configuration
	messages      []schema.Message
	maxSize       int
	callCount     int
	mu            sync.RWMutex
	shouldError   bool
	errorToReturn error
	simulateDelay time.Duration
}

// NewAdvancedMockChatMessageHistory creates a new advanced mock chat message history
func NewAdvancedMockChatMessageHistory(options ...MockHistoryOption) *AdvancedMockChatMessageHistory {
	mock := &AdvancedMockChatMessageHistory{
		messages: make([]schema.Message, 0),
		maxSize:  100,
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	return mock
}

// MockHistoryOption defines functional options for mock history configuration
type MockHistoryOption func(*AdvancedMockChatMessageHistory)

// WithHistoryMaxSize sets the maximum history size
func WithHistoryMaxSize(size int) MockHistoryOption {
	return func(h *AdvancedMockChatMessageHistory) {
		h.maxSize = size
	}
}

// WithHistoryError configures the mock to return errors
func WithHistoryError(shouldError bool, err error) MockHistoryOption {
	return func(h *AdvancedMockChatMessageHistory) {
		h.shouldError = shouldError
		h.errorToReturn = err
	}
}

// WithHistoryDelay adds artificial delay to mock operations
func WithHistoryDelay(delay time.Duration) MockHistoryOption {
	return func(h *AdvancedMockChatMessageHistory) {
		h.simulateDelay = delay
	}
}

// WithPreloadedHistoryMessages preloads messages into the mock history
func WithPreloadedHistoryMessages(messages []schema.Message) MockHistoryOption {
	return func(h *AdvancedMockChatMessageHistory) {
		h.messages = make([]schema.Message, len(messages))
		copy(h.messages, messages)
	}
}

// Mock implementation methods for ChatMessageHistory
func (h *AdvancedMockChatMessageHistory) AddMessage(ctx context.Context, message schema.Message) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.callCount++

	if h.simulateDelay > 0 {
		time.Sleep(h.simulateDelay)
	}

	if h.shouldError {
		return h.errorToReturn
	}

	h.messages = append(h.messages, message)

	// Enforce size limits
	if len(h.messages) > h.maxSize {
		h.messages = h.messages[len(h.messages)-h.maxSize:]
	}

	return nil
}

func (h *AdvancedMockChatMessageHistory) AddUserMessage(ctx context.Context, content string) error {
	return h.AddMessage(ctx, schema.NewHumanMessage(content))
}

func (h *AdvancedMockChatMessageHistory) AddAIMessage(ctx context.Context, content string) error {
	return h.AddMessage(ctx, schema.NewAIMessage(content))
}

func (h *AdvancedMockChatMessageHistory) GetMessages(ctx context.Context) ([]schema.Message, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	h.callCount++

	if h.simulateDelay > 0 {
		time.Sleep(h.simulateDelay)
	}

	if h.shouldError {
		return nil, h.errorToReturn
	}

	result := make([]schema.Message, len(h.messages))
	copy(result, h.messages)
	return result, nil
}

func (h *AdvancedMockChatMessageHistory) Clear(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.callCount++

	if h.simulateDelay > 0 {
		time.Sleep(h.simulateDelay)
	}

	if h.shouldError {
		return h.errorToReturn
	}

	h.messages = make([]schema.Message, 0)
	return nil
}

func (h *AdvancedMockChatMessageHistory) GetCallCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.callCount
}

// Test data creation helpers

// CreateTestMessages creates a set of test messages
func CreateTestMessages(count int) []schema.Message {
	messages := make([]schema.Message, 0, count*2)

	for i := 0; i < count; i++ {
		humanMsg := schema.NewHumanMessage(fmt.Sprintf("Human message %d", i+1))
		aiMsg := schema.NewAIMessage(fmt.Sprintf("AI response %d", i+1))

		messages = append(messages, humanMsg, aiMsg)
	}

	return messages
}

// CreateTestMemoryConfig creates a test memory configuration
func CreateTestMemoryConfig(memoryType MemoryType) Config {
	return Config{
		Type:           memoryType,
		MemoryKey:      "test_history",
		InputKey:       "input",
		OutputKey:      "output",
		ReturnMessages: false,
		WindowSize:     5,
		MaxTokenLimit:  2000,
		TopK:           4,
		HumanPrefix:    "Human",
		AIPrefix:       "AI",
		Enabled:        true,
		Timeout:        30 * time.Second,
	}
}

// CreateTestInputOutput creates test input/output maps
func CreateTestInputOutput(input, output string) (map[string]any, map[string]any) {
	inputs := map[string]any{
		"input": input,
	}
	outputs := map[string]any{
		"output": output,
	}
	return inputs, outputs
}

// Assertion helpers

// AssertMemoryVariables validates memory variables
func AssertMemoryVariables(t *testing.T, memory iface.Memory, expectedVars []string) {
	variables := memory.MemoryVariables()
	assert.ElementsMatch(t, expectedVars, variables)
}

// AssertMemoryContent validates memory content
func AssertMemoryContent(t *testing.T, content map[string]any, expectedKeys []string) {
	for _, key := range expectedKeys {
		assert.Contains(t, content, key)
		assert.NotEmpty(t, content[key])
	}
}

// AssertMessageHistory validates message history
func AssertMessageHistory(t *testing.T, messages []schema.Message, expectedCount int) {
	assert.Len(t, messages, expectedCount)

	for _, msg := range messages {
		assert.NotEmpty(t, msg.GetContent())
		msgType := msg.GetType()
		assert.True(t, msgType == schema.RoleHuman || msgType == schema.RoleAssistant || msgType == schema.RoleSystem,
			"message type should be human, assistant, or system, got: %s", msgType)
	}
}

// AssertHealthCheck validates health check results
func AssertHealthCheck(t *testing.T, health map[string]interface{}, expectedStatus string) {
	assert.Contains(t, health, "status")
	assert.Equal(t, expectedStatus, health["status"])
	assert.Contains(t, health, "memory_key")
	assert.Contains(t, health, "memory_type")
}

// AssertErrorType validates error types and codes
func AssertErrorType(t *testing.T, err error, expectedCode string) {
	assert.Error(t, err)
	var memErr *MemoryError
	if assert.ErrorAs(t, err, &memErr) {
		assert.Equal(t, expectedCode, memErr.Code)
	}
}

// Performance testing helpers

// ConcurrentTestRunner runs memory tests concurrently for performance testing
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

// RunLoadTest executes a load test scenario on memory
func RunLoadTest(t *testing.T, memory *AdvancedMockMemory, numOperations int, concurrency int) {
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

			// Simulate save context operation
			inputs, outputs := CreateTestInputOutput(
				fmt.Sprintf("input-%d", opID),
				fmt.Sprintf("output-%d", opID),
			)

			err := memory.SaveContext(ctx, inputs, outputs)
			if err != nil {
				errChan <- err
				return
			}

			// Simulate load memory variables operation
			_, err = memory.LoadMemoryVariables(ctx, inputs)
			if err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verify no errors occurred
	for err := range errChan {
		assert.NoError(t, err)
	}

	// Verify expected operation count (each iteration does 2 operations)
	assert.Equal(t, numOperations*2, memory.GetCallCount())
}

// Integration test helpers

// IntegrationTestHelper provides utilities for integration testing
type IntegrationTestHelper struct {
	memories  map[string]*AdvancedMockMemory
	histories map[string]*AdvancedMockChatMessageHistory
}

func NewIntegrationTestHelper() *IntegrationTestHelper {
	return &IntegrationTestHelper{
		memories:  make(map[string]*AdvancedMockMemory),
		histories: make(map[string]*AdvancedMockChatMessageHistory),
	}
}

func (h *IntegrationTestHelper) AddMemory(name string, memory *AdvancedMockMemory) {
	h.memories[name] = memory
}

func (h *IntegrationTestHelper) AddHistory(name string, history *AdvancedMockChatMessageHistory) {
	h.histories[name] = history
}

func (h *IntegrationTestHelper) GetMemory(name string) *AdvancedMockMemory {
	return h.memories[name]
}

func (h *IntegrationTestHelper) GetHistory(name string) *AdvancedMockChatMessageHistory {
	return h.histories[name]
}

func (h *IntegrationTestHelper) Reset() {
	for _, memory := range h.memories {
		memory.Clear(context.Background())
		memory.callCount = 0
	}
	for _, history := range h.histories {
		history.Clear(context.Background())
		history.callCount = 0
	}
}

// MemoryScenarioRunner runs common memory scenarios
type MemoryScenarioRunner struct {
	memory iface.Memory
}

func NewMemoryScenarioRunner(memory iface.Memory) *MemoryScenarioRunner {
	return &MemoryScenarioRunner{memory: memory}
}

func (r *MemoryScenarioRunner) RunConversationScenario(ctx context.Context, exchanges int) error {
	for i := 0; i < exchanges; i++ {
		inputs, outputs := CreateTestInputOutput(
			fmt.Sprintf("Question %d: What is AI?", i+1),
			fmt.Sprintf("Answer %d: AI is artificial intelligence.", i+1),
		)

		// Load current memory
		_, err := r.memory.LoadMemoryVariables(ctx, inputs)
		if err != nil {
			return fmt.Errorf("failed to load memory variables: %w", err)
		}

		// Save new context
		err = r.memory.SaveContext(ctx, inputs, outputs)
		if err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}
	}

	return nil
}

func (r *MemoryScenarioRunner) RunMemoryRetentionTest(ctx context.Context, initialSize, targetSize int) error {
	// Fill memory beyond capacity
	for i := 0; i < initialSize; i++ {
		inputs, outputs := CreateTestInputOutput(
			fmt.Sprintf("Input %d", i),
			fmt.Sprintf("Output %d", i),
		)

		err := r.memory.SaveContext(ctx, inputs, outputs)
		if err != nil {
			return err
		}
	}

	// Check final memory state
	vars, err := r.memory.LoadMemoryVariables(ctx, map[string]any{"input": "test"})
	if err != nil {
		return err
	}

	// Verify memory was properly managed (implementation-specific)
	if len(vars) == 0 {
		return fmt.Errorf("memory appears to be empty after retention test")
	}

	return nil
}
