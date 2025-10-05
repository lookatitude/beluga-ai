// Package schema provides advanced test utilities and comprehensive mocks for testing schema implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package schema

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockMessage provides a comprehensive mock implementation for testing
type AdvancedMockMessage struct {
	mock.Mock

	// Configuration
	messageType iface.MessageType
	content     string
	callCount   int
	mu          sync.RWMutex

	// Message metadata
	toolCalls      []iface.ToolCall
	additionalArgs map[string]interface{}

	// Health check data
	healthState     string
	lastHealthCheck time.Time
}

// NewAdvancedMockMessage creates a new advanced mock message
func NewAdvancedMockMessage(messageType iface.MessageType, content string, options ...MockMessageOption) *AdvancedMockMessage {
	mock := &AdvancedMockMessage{
		messageType:    messageType,
		content:        content,
		toolCalls:      []iface.ToolCall{},
		additionalArgs: make(map[string]interface{}),
		healthState:    "healthy",
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	return mock
}

// MockMessageOption defines functional options for mock configuration
type MockMessageOption func(*AdvancedMockMessage)

// WithMockToolCalls sets tool calls for the mock message
func WithMockToolCalls(toolCalls []iface.ToolCall) MockMessageOption {
	return func(m *AdvancedMockMessage) {
		m.toolCalls = make([]iface.ToolCall, len(toolCalls))
		copy(m.toolCalls, toolCalls)
	}
}

// WithMockAdditionalArgs sets additional arguments for the mock message
func WithMockAdditionalArgs(args map[string]interface{}) MockMessageOption {
	return func(m *AdvancedMockMessage) {
		m.additionalArgs = make(map[string]interface{})
		for k, v := range args {
			m.additionalArgs[k] = v
		}
	}
}

// Mock implementation methods for Message interface
func (m *AdvancedMockMessage) GetType() iface.MessageType {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()
	return m.messageType
}

func (m *AdvancedMockMessage) GetContent() string {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()
	return m.content
}

func (m *AdvancedMockMessage) ToolCalls() []iface.ToolCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]iface.ToolCall, len(m.toolCalls))
	copy(result, m.toolCalls)
	return result
}

func (m *AdvancedMockMessage) AdditionalArgs() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]interface{})
	for k, v := range m.additionalArgs {
		result[k] = v
	}
	return result
}

// Additional helper methods for testing
func (m *AdvancedMockMessage) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

func (m *AdvancedMockMessage) SetContent(content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.content = content
}

func (m *AdvancedMockMessage) AddToolCall(toolCall iface.ToolCall) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolCalls = append(m.toolCalls, toolCall)
}

func (m *AdvancedMockMessage) CheckHealth() map[string]interface{} {
	m.lastHealthCheck = time.Now()
	return map[string]interface{}{
		"status":          m.healthState,
		"type":            string(m.messageType),
		"content_length":  len(m.content),
		"tool_calls":      len(m.toolCalls),
		"additional_args": len(m.additionalArgs),
		"call_count":      m.callCount,
		"last_checked":    m.lastHealthCheck,
	}
}

// AdvancedMockDocument provides a comprehensive mock implementation for testing
type AdvancedMockDocument struct {
	pageContent string
	metadata    map[string]string
	id          string
	embedding   []float32
	score       float32
	callCount   int
	mu          sync.RWMutex
}

func NewAdvancedMockDocument(content string, metadata map[string]string, options ...MockDocumentOption) *AdvancedMockDocument {
	mock := &AdvancedMockDocument{
		pageContent: content,
		metadata:    make(map[string]string),
		embedding:   []float32{},
		score:       0.0,
	}

	// Copy metadata
	for k, v := range metadata {
		mock.metadata[k] = v
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	return mock
}

// MockDocumentOption defines functional options for mock document configuration
type MockDocumentOption func(*AdvancedMockDocument)

// WithMockEmbedding sets the embedding for the mock document
func WithMockEmbedding(embedding []float32) MockDocumentOption {
	return func(d *AdvancedMockDocument) {
		d.embedding = make([]float32, len(embedding))
		copy(d.embedding, embedding)
	}
}

// WithMockScore sets the score for the mock document
func WithMockScore(score float32) MockDocumentOption {
	return func(d *AdvancedMockDocument) {
		d.score = score
	}
}

// WithMockID sets the ID for the mock document
func WithMockID(id string) MockDocumentOption {
	return func(d *AdvancedMockDocument) {
		d.id = id
	}
}

// Document interface implementation
func (d *AdvancedMockDocument) GetContent() string {
	d.mu.Lock()
	d.callCount++
	d.mu.Unlock()
	return d.pageContent
}

func (d *AdvancedMockDocument) GetType() iface.MessageType {
	return iface.RoleSystem // Documents are typically system messages
}

func (d *AdvancedMockDocument) ToolCalls() []iface.ToolCall {
	return []iface.ToolCall{} // Documents don't have tool calls
}

func (d *AdvancedMockDocument) AdditionalArgs() map[string]interface{} {
	return make(map[string]interface{}) // Documents don't have additional args
}

// Additional document-specific methods
func (d *AdvancedMockDocument) GetMetadata() map[string]string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range d.metadata {
		result[k] = v
	}
	return result
}

func (d *AdvancedMockDocument) GetID() string {
	return d.id
}

func (d *AdvancedMockDocument) GetEmbedding() []float32 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make([]float32, len(d.embedding))
	copy(result, d.embedding)
	return result
}

func (d *AdvancedMockDocument) GetScore() float32 {
	return d.score
}

func (d *AdvancedMockDocument) GetCallCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.callCount
}

// Test data creation helpers

// CreateTestMessages creates a variety of test messages
func CreateTestMessages(count int) []Message {
	messages := make([]Message, count)
	messageTypes := []iface.MessageType{iface.RoleHuman, iface.RoleAssistant, iface.RoleSystem}

	for i := 0; i < count; i++ {
		msgType := messageTypes[i%len(messageTypes)]
		content := fmt.Sprintf("Test message %d of type %s", i+1, msgType)

		switch msgType {
		case iface.RoleHuman:
			messages[i] = NewHumanMessage(content)
		case iface.RoleAssistant:
			messages[i] = NewAIMessage(content)
		case iface.RoleSystem:
			messages[i] = NewSystemMessage(content)
		}
	}

	return messages
}

// CreateTestDocuments creates a set of test documents
func CreateTestDocuments(count int, topic string) []Document {
	documents := make([]Document, count)

	for i := 0; i < count; i++ {
		content := fmt.Sprintf("Test document %d about %s. This document contains comprehensive information about %s for testing purposes.",
			i+1, topic, topic)
		metadata := map[string]string{
			"doc_id":   fmt.Sprintf("test_doc_%d", i+1),
			"topic":    topic,
			"index":    fmt.Sprintf("%d", i+1),
			"category": fmt.Sprintf("category_%d", (i%3)+1),
		}

		documents[i] = NewDocument(content, metadata)
	}

	return documents
}

// CreateTestToolCall creates a test tool call
func CreateTestToolCall(name, input string) iface.ToolCall {
	return iface.ToolCall{
		ID:        fmt.Sprintf("call_%s_%d", name, time.Now().UnixNano()),
		Type:      "function",
		Name:      name,
		Arguments: fmt.Sprintf(`{"input": "%s"}`, input),
		Function: iface.FunctionCall{
			Name:      name,
			Arguments: fmt.Sprintf(`{"input": "%s"}`, input),
		},
	}
}

// CreateTestAgentAction creates a test agent action
func CreateTestAgentAction(tool, input, log string) AgentAction {
	return AgentAction{
		Tool:      tool,
		ToolInput: input,
		Log:       log,
	}
}

// CreateTestAgentFinish creates a test agent finish
func CreateTestAgentFinish(returnValues map[string]any, log string) AgentFinish {
	return AgentFinish{
		ReturnValues: returnValues,
		Log:          log,
	}
}

// Assertion helpers

// AssertMessage validates message properties
func AssertMessage(t *testing.T, message Message, expectedType iface.MessageType, expectedMinLength int) {
	assert.Equal(t, expectedType, message.GetType(), "Message type should match")
	assert.GreaterOrEqual(t, len(message.GetContent()), expectedMinLength, "Message content should have minimum length")
	assert.NotNil(t, message.ToolCalls(), "Tool calls should not be nil (can be empty)")
	assert.NotNil(t, message.AdditionalArgs(), "Additional args should not be nil (can be empty)")
}

// AssertDocument validates document properties
func AssertDocument(t *testing.T, document Document, expectedMinContentLength int) {
	assert.GreaterOrEqual(t, len(document.GetContent()), expectedMinContentLength, "Document should have substantial content")
	assert.NotNil(t, document.Metadata, "Document should have metadata")
	assert.Equal(t, iface.RoleSystem, document.GetType(), "Document should be system type message")
}

// AssertToolCall validates tool call properties
func AssertToolCall(t *testing.T, toolCall iface.ToolCall, expectedName string) {
	assert.Equal(t, expectedName, toolCall.Name, "Tool call name should match")
	assert.NotEmpty(t, toolCall.ID, "Tool call should have ID")
	assert.NotEmpty(t, toolCall.Arguments, "Tool call should have arguments")
	assert.Equal(t, "function", toolCall.Type, "Tool call should be function type")
}

// AssertAgentAction validates agent action properties
func AssertAgentAction(t *testing.T, action AgentAction, expectedTool string) {
	assert.Equal(t, expectedTool, action.Tool, "Agent action tool should match")
	assert.NotNil(t, action.ToolInput, "Agent action should have tool input")
	assert.NotEmpty(t, action.Log, "Agent action should have log")
}

// AssertAgentFinish validates agent finish properties
func AssertAgentFinish(t *testing.T, finish AgentFinish, expectedMinReturnValues int) {
	assert.GreaterOrEqual(t, len(finish.ReturnValues), expectedMinReturnValues,
		"Agent finish should have minimum return values")
	assert.NotEmpty(t, finish.Log, "Agent finish should have log")
}

// AssertMessageHistory validates message history
func AssertMessageHistory(t *testing.T, messages []Message, expectedMinCount int) {
	assert.GreaterOrEqual(t, len(messages), expectedMinCount, "History should have minimum message count")

	for i, msg := range messages {
		assert.NotNil(t, msg, "Message %d should not be nil", i)
		assert.NotEmpty(t, msg.GetContent(), "Message %d should have content", i)
	}
}

// AssertErrorType validates error types and codes
func AssertErrorType(t *testing.T, err error, expectedCode string) {
	assert.Error(t, err)
	var schemaErr *iface.SchemaError
	if assert.ErrorAs(t, err, &schemaErr) {
		assert.Equal(t, expectedCode, schemaErr.Code)
	}
}

// Performance testing helpers

// ConcurrentTestRunner runs schema tests concurrently for performance testing
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

// RunLoadTest executes a load test scenario on schema components
func RunLoadTest(t *testing.T, message *AdvancedMockMessage, numOperations int, concurrency int) {
	var wg sync.WaitGroup
	errChan := make(chan error, numOperations)

	semaphore := make(chan struct{}, concurrency)

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(opID int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Test various message operations
			if opID%4 == 0 {
				_ = message.GetType()
			} else if opID%4 == 1 {
				_ = message.GetContent()
			} else if opID%4 == 2 {
				_ = message.ToolCalls()
			} else {
				_ = message.AdditionalArgs()
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verify no errors occurred
	for err := range errChan {
		assert.NoError(t, err)
	}

	// Verify expected call count (GetType and GetContent increment counter)
	expectedMinCalls := numOperations / 2
	assert.GreaterOrEqual(t, message.GetCallCount(), expectedMinCalls)
}

// Integration test helpers

// IntegrationTestHelper provides utilities for integration testing
type IntegrationTestHelper struct {
	messages  map[string]*AdvancedMockMessage
	documents map[string]*AdvancedMockDocument
}

func NewIntegrationTestHelper() *IntegrationTestHelper {
	return &IntegrationTestHelper{
		messages:  make(map[string]*AdvancedMockMessage),
		documents: make(map[string]*AdvancedMockDocument),
	}
}

func (h *IntegrationTestHelper) AddMessage(name string, message *AdvancedMockMessage) {
	h.messages[name] = message
}

func (h *IntegrationTestHelper) AddDocument(name string, document *AdvancedMockDocument) {
	h.documents[name] = document
}

func (h *IntegrationTestHelper) GetMessage(name string) *AdvancedMockMessage {
	return h.messages[name]
}

func (h *IntegrationTestHelper) GetDocument(name string) *AdvancedMockDocument {
	return h.documents[name]
}

func (h *IntegrationTestHelper) Reset() {
	for _, message := range h.messages {
		message.callCount = 0
	}
	for _, document := range h.documents {
		document.callCount = 0
	}
}

// SchemaScenarioRunner runs common schema scenarios
type SchemaScenarioRunner struct {
	messages  []Message
	documents []Document
}

func NewSchemaScenarioRunner(messageCount, documentCount int) *SchemaScenarioRunner {
	return &SchemaScenarioRunner{
		messages:  CreateTestMessages(messageCount),
		documents: CreateTestDocuments(documentCount, "test"),
	}
}

func (r *SchemaScenarioRunner) RunMessageProcessingScenario() error {
	for i, message := range r.messages {
		// Test message operations
		msgType := message.GetType()
		content := message.GetContent()
		toolCalls := message.ToolCalls()
		additionalArgs := message.AdditionalArgs()

		// Validate message properties
		if msgType == "" {
			return fmt.Errorf("message %d has empty type", i+1)
		}
		if content == "" {
			return fmt.Errorf("message %d has empty content", i+1)
		}
		if toolCalls == nil {
			return fmt.Errorf("message %d has nil tool calls", i+1)
		}
		if additionalArgs == nil {
			return fmt.Errorf("message %d has nil additional args", i+1)
		}
	}

	return nil
}

func (r *SchemaScenarioRunner) RunDocumentProcessingScenario() error {
	for i, document := range r.documents {
		// Test document operations
		content := document.GetContent()
		msgType := document.GetType()
		metadata := document.Metadata

		// Validate document properties
		if content == "" {
			return fmt.Errorf("document %d has empty content", i+1)
		}
		if msgType != iface.RoleSystem {
			return fmt.Errorf("document %d should be system type, got %s", i+1, msgType)
		}
		if metadata == nil {
			return fmt.Errorf("document %d has nil metadata", i+1)
		}
	}

	return nil
}

func (r *SchemaScenarioRunner) RunConversationScenario() error {
	// Test conversation flow with alternating message types
	conversation := make([]Message, 0)

	// Build conversation
	for i := 0; i < len(r.messages)/2; i++ {
		if i*2 < len(r.messages) {
			conversation = append(conversation, r.messages[i*2])
		}
		if i*2+1 < len(r.messages) {
			conversation = append(conversation, r.messages[i*2+1])
		}
	}

	// Validate conversation flow
	for i, msg := range conversation {
		if i%2 == 0 {
			// Should be human message
			if msg.GetType() != iface.RoleHuman {
				return fmt.Errorf("conversation message %d should be human, got %s", i+1, msg.GetType())
			}
		} else {
			// Should be AI message
			if msg.GetType() != iface.RoleAssistant {
				return fmt.Errorf("conversation message %d should be assistant, got %s", i+1, msg.GetType())
			}
		}
	}

	return nil
}

// BenchmarkHelper provides benchmarking utilities for schema
type BenchmarkHelper struct {
	messages  []Message
	documents []Document
}

func NewBenchmarkHelper(messageCount, documentCount int) *BenchmarkHelper {
	return &BenchmarkHelper{
		messages:  CreateTestMessages(messageCount),
		documents: CreateTestDocuments(documentCount, "benchmark"),
	}
}

func (b *BenchmarkHelper) BenchmarkMessageOperations(iterations int) (time.Duration, error) {
	start := time.Now()
	for i := 0; i < iterations; i++ {
		message := b.messages[i%len(b.messages)]

		// Test all message operations
		_ = message.GetType()
		_ = message.GetContent()
		_ = message.ToolCalls()
		_ = message.AdditionalArgs()
	}

	return time.Since(start), nil
}

func (b *BenchmarkHelper) BenchmarkDocumentOperations(iterations int) (time.Duration, error) {
	start := time.Now()
	for i := 0; i < iterations; i++ {
		document := b.documents[i%len(b.documents)]

		// Test all document operations
		_ = document.GetContent()
		_ = document.GetType()
		_ = document.Metadata
	}

	return time.Since(start), nil
}

func (b *BenchmarkHelper) BenchmarkSchemaCreation(iterations int) (time.Duration, error) {
	start := time.Now()
	for i := 0; i < iterations; i++ {
		// Benchmark creating new schema objects
		_ = NewHumanMessage(fmt.Sprintf("Benchmark message %d", i))
		_ = NewDocument(fmt.Sprintf("Benchmark document %d", i), map[string]string{"id": fmt.Sprintf("%d", i)})
	}

	return time.Since(start), nil
}

// Schema validation helpers

// ValidateMessageType checks if a message type is valid
func ValidateMessageType(msgType iface.MessageType) bool {
	validTypes := []iface.MessageType{
		iface.RoleHuman,
		iface.RoleAssistant,
		iface.RoleSystem,
		iface.RoleTool,
		iface.RoleFunction,
	}

	for _, validType := range validTypes {
		if msgType == validType {
			return true
		}
	}

	return false
}

// ValidateConversationFlow checks if message sequence follows proper conversation patterns
func ValidateConversationFlow(messages []Message) error {
	if len(messages) == 0 {
		return nil // Empty conversation is valid
	}

	// Check for basic conversation patterns
	for i, message := range messages {
		msgType := message.GetType()

		// Validate message type
		if !ValidateMessageType(msgType) {
			return fmt.Errorf("message %d has invalid type: %s", i+1, msgType)
		}

		// Check content is not empty
		if message.GetContent() == "" {
			return fmt.Errorf("message %d has empty content", i+1)
		}
	}

	return nil
}

// ValidateDocumentCollection validates a collection of documents
func ValidateDocumentCollection(documents []Document) error {
	for i, doc := range documents {
		if doc.GetContent() == "" {
			return fmt.Errorf("document %d has empty content", i+1)
		}

		if doc.Metadata == nil {
			return fmt.Errorf("document %d has nil metadata", i+1)
		}

		if doc.GetType() != iface.RoleSystem {
			return fmt.Errorf("document %d should be system type, got %s", i+1, doc.GetType())
		}
	}

	return nil
}
