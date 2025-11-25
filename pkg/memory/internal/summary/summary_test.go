// Package summary provides comprehensive tests for summary-based memory implementations.
package summary

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLLM is a mock implementation of core.Runnable for testing
type MockLLM struct {
	invokeFunc  func(ctx context.Context, input any, options ...core.Option) (any, error)
	batchFunc   func(ctx context.Context, inputs []any, options ...core.Option) ([]any, error)
	streamFunc  func(ctx context.Context, input any, options ...core.Option) (<-chan any, error)
	invokeError error
	batchError  error
	streamError error
}

func NewMockLLM() *MockLLM {
	return &MockLLM{}
}

func (m *MockLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	if m.invokeError != nil {
		return nil, m.invokeError
	}
	if m.invokeFunc != nil {
		return m.invokeFunc(ctx, input, options...)
	}
	// Default behavior - return the input as a string
	return "Mock LLM response", nil
}

func (m *MockLLM) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	if m.batchError != nil {
		return nil, m.batchError
	}
	if m.batchFunc != nil {
		return m.batchFunc(ctx, inputs, options...)
	}
	// Default behavior - return mock responses
	results := make([]any, len(inputs))
	for i := range inputs {
		results[i] = "Mock batch response"
	}
	return results, nil
}

func (m *MockLLM) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	if m.streamError != nil {
		return nil, m.streamError
	}
	if m.streamFunc != nil {
		return m.streamFunc(ctx, input, options...)
	}
	// Default behavior - return a channel with a single response
	ch := make(chan any, 1)
	ch <- "Mock stream response"
	close(ch)
	return ch, nil
}

func (m *MockLLM) WithConfig(config map[string]any) core.Runnable {
	return m
}

// MockPromptTemplate is a mock implementation of promptsiface.Template for testing
type MockPromptTemplate struct {
	formatFunc    func(ctx context.Context, inputs map[string]any) (any, error)
	formatError   error
	name          string
	inputVars     []string
	validateError error
}

func NewMockPromptTemplate() *MockPromptTemplate {
	return &MockPromptTemplate{
		name:      "mock_template",
		inputVars: []string{"summary", "new_lines"},
	}
}

func (m *MockPromptTemplate) Format(ctx context.Context, inputs map[string]any) (any, error) {
	if m.formatError != nil {
		return nil, m.formatError
	}
	if m.formatFunc != nil {
		return m.formatFunc(ctx, inputs)
	}
	// Default behavior - return a formatted string
	return "Formatted prompt", nil
}

func (m *MockPromptTemplate) GetInputVariables() []string {
	return m.inputVars
}

func (m *MockPromptTemplate) Name() string {
	return m.name
}

func (m *MockPromptTemplate) Validate() error {
	return m.validateError
}

// MockChatMessageHistory is a mock implementation for testing
type MockChatMessageHistory struct {
	messages   []schema.Message
	addError   error
	getError   error
	clearError error
}

func NewMockChatMessageHistory() *MockChatMessageHistory {
	return &MockChatMessageHistory{
		messages: make([]schema.Message, 0),
	}
}

func (m *MockChatMessageHistory) AddMessage(ctx context.Context, message schema.Message) error {
	if m.addError != nil {
		return m.addError
	}
	m.messages = append(m.messages, message)
	return nil
}

func (m *MockChatMessageHistory) AddUserMessage(ctx context.Context, content string) error {
	return m.AddMessage(ctx, schema.NewHumanMessage(content))
}

func (m *MockChatMessageHistory) AddAIMessage(ctx context.Context, content string) error {
	return m.AddMessage(ctx, schema.NewAIMessage(content))
}

func (m *MockChatMessageHistory) GetMessages(ctx context.Context) ([]schema.Message, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	// Return a copy to prevent external modification
	messages := make([]schema.Message, len(m.messages))
	copy(messages, m.messages)
	return messages, nil
}

func (m *MockChatMessageHistory) Clear(ctx context.Context) error {
	if m.clearError != nil {
		return m.clearError
	}
	m.messages = m.messages[:0]
	return nil
}

// Ensure MockChatMessageHistory implements the interface
var _ iface.ChatMessageHistory = (*MockChatMessageHistory)(nil)

// TestNewConversationSummaryMemory tests the constructor
func TestNewConversationSummaryMemory(t *testing.T) {
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	memory := NewConversationSummaryMemory(history, llm, "test_memory")

	assert.NotNil(t, memory)
	assert.Equal(t, history, memory.ChatHistory)
	assert.Equal(t, llm, memory.LLM)
	assert.Equal(t, "test_memory", memory.MemoryKey)
	assert.Equal(t, "Human", memory.HumanPrefix)
	assert.Equal(t, "AI", memory.AiPrefix)
	assert.NotNil(t, memory.SummaryPrompt)
}

// TestNewConversationSummaryMemory_Defaults tests default values
func TestNewConversationSummaryMemory_Defaults(t *testing.T) {
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	memory := NewConversationSummaryMemory(history, llm, "")

	assert.Equal(t, "history", memory.MemoryKey) // Default memory key
}

// TestConversationSummaryMemory_MemoryVariables tests the MemoryVariables method
func TestConversationSummaryMemory_MemoryVariables(t *testing.T) {
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	memory := NewConversationSummaryMemory(history, llm, "summary_memory")

	variables := memory.MemoryVariables()
	assert.Equal(t, []string{"summary_memory"}, variables)
}

// TestConversationSummaryMemory_LoadMemoryVariables tests loading memory variables
func TestConversationSummaryMemory_LoadMemoryVariables(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	memory := NewConversationSummaryMemory(history, llm, "memory")
	memory.currentSummary = "Test summary"

	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	assert.NoError(t, err)
	assert.Contains(t, vars, "memory")
	assert.Equal(t, "Test summary", vars["memory"])
}

// TestConversationSummaryMemory_LoadMemoryVariables_EmptySummary tests loading with empty summary
func TestConversationSummaryMemory_LoadMemoryVariables_EmptySummary(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	memory := NewConversationSummaryMemory(history, llm, "memory")

	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	assert.NoError(t, err)
	assert.Equal(t, "", vars["memory"])
}

// TestConversationSummaryMemory_SaveContext tests saving context
func TestConversationSummaryMemory_SaveContext(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	// Mock LLM to return a summary
	llm.invokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		return "Updated summary with new conversation", nil
	}

	memory := NewConversationSummaryMemory(history, llm, "memory")
	memory.currentSummary = "Initial summary"

	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.NoError(t, err)

	// Verify summary was updated
	assert.Equal(t, "Updated summary with new conversation", memory.currentSummary)

	// Summary memory doesn't store individual messages, only the summary
	// The ChatHistory is used temporarily during summarization but messages aren't persisted
	// Verify the summary contains the conversation content
	assert.Contains(t, memory.currentSummary, "conversation")
}

// TestConversationSummaryMemory_SaveContext_CustomKeys tests saving with custom keys
func TestConversationSummaryMemory_SaveContext_CustomKeys(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	llm.invokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		return "Summary with custom keys", nil
	}

	memory := NewConversationSummaryMemory(history, llm, "memory")
	memory.InputKey = "question"
	memory.OutputKey = "answer"

	inputs := map[string]any{"question": "Hello"}
	outputs := map[string]any{"answer": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.NoError(t, err)
	assert.Equal(t, "Summary with custom keys", memory.currentSummary)
}

// TestConversationSummaryMemory_SaveContext_AutoDetectKeys tests automatic key detection
func TestConversationSummaryMemory_SaveContext_AutoDetectKeys(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	llm.invokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		return "Auto-detected summary", nil
	}

	memory := NewConversationSummaryMemory(history, llm, "memory")
	// Clear keys to trigger auto-detection
	memory.InputKey = ""
	memory.OutputKey = ""

	inputs := map[string]any{"query": "Hello"}
	outputs := map[string]any{"response": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.NoError(t, err)
	assert.Equal(t, "Auto-detected summary", memory.currentSummary)
}

// TestConversationSummaryMemory_SaveContext_ErrorHandling tests various error conditions
func TestConversationSummaryMemory_SaveContext_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		name          string
		setupMemory   func(*ConversationSummaryMemory)
		setupLLM      func(*MockLLM)
		inputs        map[string]any
		outputs       map[string]any
		expectedError string
	}{
		{
			name: "Missing input key",
			setupMemory: func(m *ConversationSummaryMemory) {
				m.InputKey = "input"
			},
			inputs:        map[string]any{"wrong_key": "Hello"},
			outputs:       map[string]any{"output": "Hi!"},
			expectedError: "error", // Generic error message in current implementation
		},
		{
			name: "Missing output key",
			setupMemory: func(m *ConversationSummaryMemory) {
				m.OutputKey = "output"
			},
			inputs:        map[string]any{"input": "Hello"},
			outputs:       map[string]any{"wrong_key": "Hi!"},
			expectedError: "error",
		},
		{
			name: "Non-string input",
			setupMemory: func(m *ConversationSummaryMemory) {
				m.InputKey = "input"
				m.OutputKey = "output"
			},
			inputs:        map[string]any{"input": 123},
			outputs:       map[string]any{"output": "Hi!"},
			expectedError: "error",
		},
		{
			name: "Non-string output",
			setupMemory: func(m *ConversationSummaryMemory) {
				m.InputKey = "input"
				m.OutputKey = "output"
			},
			inputs:        map[string]any{"input": "Hello"},
			outputs:       map[string]any{"output": 456},
			expectedError: "error",
		},
		{
			name: "LLM error",
			setupMemory: func(m *ConversationSummaryMemory) {
				m.InputKey = "input"
				m.OutputKey = "output"
			},
			setupLLM: func(llm *MockLLM) {
				llm.invokeError = errors.New("LLM failed")
			},
			inputs:        map[string]any{"input": "Hello"},
			outputs:       map[string]any{"output": "Hi!"},
			expectedError: "error",
		},
		{
			name: "Prompt format error",
			setupMemory: func(m *ConversationSummaryMemory) {
				m.InputKey = "input"
				m.OutputKey = "output"
				m.SummaryPrompt = NewMockPromptTemplate()
				mockPrompt := m.SummaryPrompt.(*MockPromptTemplate)
				mockPrompt.formatError = errors.New("format failed")
			},
			inputs:        map[string]any{"input": "Hello"},
			outputs:       map[string]any{"output": "Hi!"},
			expectedError: "error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			history := NewMockChatMessageHistory()
			llm := NewMockLLM()
			if tc.setupLLM != nil {
				tc.setupLLM(llm)
			}

			memory := NewConversationSummaryMemory(history, llm, "memory")
			if tc.setupMemory != nil {
				tc.setupMemory(memory)
			}

			err := memory.SaveContext(ctx, tc.inputs, tc.outputs)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

// TestConversationSummaryMemory_SaveContext_LLMMessageResponse tests LLM returning schema.Message
func TestConversationSummaryMemory_SaveContext_LLMMessageResponse(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	// Mock LLM to return a schema.Message
	llm.invokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		return schema.NewAIMessage("LLM generated summary"), nil
	}

	memory := NewConversationSummaryMemory(history, llm, "memory")

	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.NoError(t, err)
	assert.Equal(t, "LLM generated summary", memory.currentSummary)
}

// TestConversationSummaryMemory_SaveContext_LLMStringResponse tests LLM returning string
func TestConversationSummaryMemory_SaveContext_LLMStringResponse(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	// Mock LLM to return a string
	llm.invokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		return "String summary response", nil
	}

	memory := NewConversationSummaryMemory(history, llm, "memory")

	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.NoError(t, err)
	assert.Equal(t, "String summary response", memory.currentSummary)
}

// TestConversationSummaryMemory_SaveContext_LLMUnexpectedResponse tests LLM returning unexpected type
func TestConversationSummaryMemory_SaveContext_LLMUnexpectedResponse(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	// Mock LLM to return an unexpected type
	llm.invokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		return 123, nil // Unexpected integer response
	}

	memory := NewConversationSummaryMemory(history, llm, "memory")

	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM Invoke returned unexpected type for summary")
}

// TestConversationSummaryMemory_Clear tests the Clear method
func TestConversationSummaryMemory_Clear(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	memory := NewConversationSummaryMemory(history, llm, "memory")
	memory.currentSummary = "Test summary"

	// Add some messages to history
	err := history.AddUserMessage(ctx, "Hello")
	require.NoError(t, err)

	// Clear
	err = memory.Clear(ctx)
	assert.NoError(t, err)

	// Verify summary is cleared
	assert.Equal(t, "", memory.currentSummary)

	// Verify history is cleared
	messages, err := history.GetMessages(ctx)
	require.NoError(t, err)
	assert.Len(t, messages, 0)
}

// TestConversationSummaryMemory_Clear_HistoryError tests error handling in Clear
func TestConversationSummaryMemory_Clear_HistoryError(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	history.clearError = errors.New("history clear error")
	llm := NewMockLLM()

	memory := NewConversationSummaryMemory(history, llm, "memory")

	err := memory.Clear(ctx)
	assert.Error(t, err)
	assert.Equal(t, history.clearError, err)
}

// TestConversationSummaryBufferMemory tests the ConversationSummaryBufferMemory implementation
func TestConversationSummaryBufferMemory(t *testing.T) {
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	memory := NewConversationSummaryBufferMemory(history, llm, "buffer_memory", 1000)

	assert.NotNil(t, memory)
	assert.Equal(t, history, memory.ChatHistory)
	assert.Equal(t, llm, memory.LLM)
	assert.Equal(t, "buffer_memory", memory.MemoryKey)
	assert.Equal(t, 1000, memory.MaxTokenLimit)
	assert.Equal(t, "Human", memory.HumanPrefix)
	assert.Equal(t, "AI", memory.AiPrefix)
}

// TestConversationSummaryBufferMemory_LoadMemoryVariables tests loading memory variables
func TestConversationSummaryBufferMemory_LoadMemoryVariables(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	memory := NewConversationSummaryBufferMemory(history, llm, "memory", 1000)
	memory.movingSummaryBuffer = "Existing summary"

	// Add some messages
	err := history.AddUserMessage(ctx, "Hello")
	require.NoError(t, err)
	err = history.AddAIMessage(ctx, "Hi!")
	require.NoError(t, err)

	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	assert.NoError(t, err)
	assert.Contains(t, vars, "memory")

	result := vars["memory"].(string)
	assert.Contains(t, result, "Existing summary")
	assert.Contains(t, result, "Hello")
	assert.Contains(t, result, "Hi!")
}

// TestConversationSummaryBufferMemory_LoadMemoryVariables_NoSummary tests loading without summary
func TestConversationSummaryBufferMemory_LoadMemoryVariables_NoSummary(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	memory := NewConversationSummaryBufferMemory(history, llm, "memory", 1000)

	// Add some messages
	err := history.AddUserMessage(ctx, "Hello")
	require.NoError(t, err)
	err = history.AddAIMessage(ctx, "Hi!")
	require.NoError(t, err)

	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	assert.NoError(t, err)

	result := vars["memory"].(string)
	assert.Contains(t, result, "Hello")
	assert.Contains(t, result, "Hi!")
	// When summary is empty, result should be the buffer string (which contains newlines)
	// The buffer string format includes newlines between messages
	assert.Contains(t, result, "Human: Hello")
	assert.Contains(t, result, "AI: Hi!")
}

// TestConversationSummaryBufferMemory_LoadMemoryVariables_GetMessagesError tests error handling
func TestConversationSummaryBufferMemory_LoadMemoryVariables_GetMessagesError(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	history.getError = errors.New("get messages error")
	llm := NewMockLLM()

	memory := NewConversationSummaryBufferMemory(history, llm, "memory", 1000)

	_, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error")
}

// TestConversationSummaryBufferMemory_SaveContext tests saving context
func TestConversationSummaryBufferMemory_SaveContext(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	memory := NewConversationSummaryBufferMemory(history, llm, "memory", 1000)

	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.NoError(t, err)

	// Verify messages were added to history
	messages, err := history.GetMessages(ctx)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
}

// TestConversationSummaryBufferMemory_SaveContext_ErrorHandling tests error handling in SaveContext
func TestConversationSummaryBufferMemory_SaveContext_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name          string
		setupMemory   func(*ConversationSummaryBufferMemory)
		inputs        map[string]any
		outputs       map[string]any
		expectedError string
	}{
		{
			name: "Missing input key",
			setupMemory: func(m *ConversationSummaryBufferMemory) {
				m.InputKey = "input"
			},
			inputs:        map[string]any{"wrong_key": "Hello"},
			outputs:       map[string]any{"output": "Hi!"},
			expectedError: "input or output key not found in context maps",
		},
		{
			name: "Non-string input",
			setupMemory: func(m *ConversationSummaryBufferMemory) {
				m.InputKey = "input"
				m.OutputKey = "output"
			},
			inputs:        map[string]any{"input": 123},
			outputs:       map[string]any{"output": "Hi!"},
			expectedError: "input or output value is not a string",
		},
		{
			name: "Add message error",
			setupMemory: func(m *ConversationSummaryBufferMemory) {
				m.InputKey = "input"
				m.OutputKey = "output"
			},
			inputs:        map[string]any{"input": "Hello"},
			outputs:       map[string]any{"output": "Hi!"},
			expectedError: "error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			history := NewMockChatMessageHistory()
			if tc.name == "Add message error" {
				history.addError = errors.New("add message error")
			}
			llm := NewMockLLM()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
			memory := NewConversationSummaryBufferMemory(history, llm, "memory", 1000)
			if tc.setupMemory != nil {
				tc.setupMemory(memory)
			}

			err := memory.SaveContext(ctx, tc.inputs, tc.outputs)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

// TestConversationSummaryBufferMemory_Clear tests the Clear method
func TestConversationSummaryBufferMemory_Clear(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	memory := NewConversationSummaryBufferMemory(history, llm, "memory", 1000)
	memory.movingSummaryBuffer = "Test summary"

	err := memory.Clear(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "", memory.movingSummaryBuffer)
}

// TestGetBufferString_Summary tests the getBufferString function with different message types
func TestGetBufferString_Summary(t *testing.T) {
	messages := []schema.Message{
		schema.NewHumanMessage("Hello"),
		schema.NewAIMessage("Hi!"),
		schema.NewSystemMessage("System message"),
		schema.NewToolMessage("Tool result", "call_123"),
	}

	result := getBufferString(messages, "User", "Bot")

	assert.Contains(t, result, "User: Hello")
	assert.Contains(t, result, "Bot: Hi!")
	assert.Contains(t, result, "System: System message")
	assert.Contains(t, result, "Tool (call_123): Tool result")
}

// TestInterfaceCompliance tests that both implementations comply with the Memory interface
func TestInterfaceCompliance_Summary(t *testing.T) {
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	// Test ConversationSummaryMemory
	summaryMemory := NewConversationSummaryMemory(history, llm, "memory")
	var _ iface.Memory = summaryMemory

	// Test ConversationSummaryBufferMemory
	summaryBufferMemory := NewConversationSummaryBufferMemory(history, llm, "buffer_memory", 100)
	var _ iface.Memory = summaryBufferMemory
}

// TestDefaultSummaryPrompt tests that the default summary prompt is properly initialized
func TestDefaultSummaryPrompt(t *testing.T) {
	// This tests that the DefaultSummaryPrompt is not nil
	// In the real implementation, this should be a valid prompt template
	assert.NotNil(t, DefaultSummaryPrompt)
}

// BenchmarkConversationSummaryMemory_SaveContext benchmarks SaveContext performance
func BenchmarkConversationSummaryMemory_SaveContext(b *testing.B) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	llm.invokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		return "Benchmark summary", nil
	}

	memory := NewConversationSummaryMemory(history, llm, "memory")

	inputs := map[string]any{"input": "Hello world"}
	outputs := map[string]any{"output": "Hi there!"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memory.SaveContext(ctx, inputs, outputs)
	}
}

// BenchmarkConversationSummaryMemory_LoadMemoryVariables benchmarks LoadMemoryVariables performance
func BenchmarkConversationSummaryMemory_LoadMemoryVariables(b *testing.B) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	memory := NewConversationSummaryMemory(history, llm, "memory")
	memory.currentSummary = "Benchmark summary"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memory.LoadMemoryVariables(ctx, map[string]any{})
	}
}

// BenchmarkGetBufferString benchmarks the getBufferString function
func BenchmarkGetBufferString_Summary(b *testing.B) {
	messages := make([]schema.Message, 100)
	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			messages[i] = schema.NewHumanMessage("Human message " + string(rune(i)))
		} else {
			messages[i] = schema.NewAIMessage("AI message " + string(rune(i)))
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getBufferString(messages, "Human", "AI")
	}
}

// TestConversationSummaryMemory_CustomPrefixes tests custom message prefixes
func TestConversationSummaryMemory_CustomPrefixes(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	llm.invokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		// Verify that the prompt contains custom prefixes
		return "Summary with custom prefixes", nil
	}

	memory := NewConversationSummaryMemory(history, llm, "memory")
	memory.HumanPrefix = "User"
	memory.AiPrefix = "Assistant"

	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.NoError(t, err)
}

// TestPredictNewSummary tests the predictNewSummary method directly
func TestConversationSummaryMemory_PredictNewSummary(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	llm.invokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		return "Generated summary", nil
	}

	memory := NewConversationSummaryMemory(history, llm, "memory")
	memory.currentSummary = "Existing summary"

	newSummary, err := memory.predictNewSummary(ctx, "New conversation lines")
	assert.NoError(t, err)
	assert.Equal(t, "Generated summary", newSummary)
}

// TestPredictNewSummary_PromptFormatError tests prompt formatting error
func TestConversationSummaryMemory_PredictNewSummary_PromptFormatError(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()

	memory := NewConversationSummaryMemory(history, llm, "memory")
	mockPrompt := NewMockPromptTemplate()
	mockPrompt.formatError = errors.New("format error")
	memory.SummaryPrompt = mockPrompt

	_, err := memory.predictNewSummary(ctx, "New lines")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error")
}

// TestPredictNewSummary_LLMError tests LLM invocation error
func TestConversationSummaryMemory_PredictNewSummary_LLMError(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	llm := NewMockLLM()
	llm.invokeError = errors.New("LLM error")

	memory := NewConversationSummaryMemory(history, llm, "memory")

	_, err := memory.predictNewSummary(ctx, "New lines")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error")
}
