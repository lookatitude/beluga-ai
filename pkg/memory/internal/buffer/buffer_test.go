// Package buffer provides comprehensive tests for buffer memory implementations.
package buffer

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockChatMessageHistory is a mock implementation for testing
type MockChatMessageHistory struct {
	messages       []schema.Message
	addError       error
	getError       error
	clearError     error
	addMessageFunc func(ctx context.Context, message schema.Message) error
}

func NewMockChatMessageHistory() *MockChatMessageHistory {
	return &MockChatMessageHistory{
		messages: make([]schema.Message, 0),
	}
}

func (m *MockChatMessageHistory) AddMessage(ctx context.Context, message schema.Message) error {
	if m.addMessageFunc != nil {
		return m.addMessageFunc(ctx, message)
	}
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

// TestNewChatMessageBufferMemory tests the constructor
func TestNewChatMessageBufferMemory(t *testing.T) {
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	assert.NotNil(t, memory)
	assert.Equal(t, history, memory.ChatHistory)
	assert.True(t, memory.ReturnMessages)
	assert.Equal(t, "history", memory.MemoryKey)
	assert.Equal(t, "input", memory.InputKey)
	assert.Equal(t, "output", memory.OutputKey)
	assert.Equal(t, "Human", memory.HumanPrefix)
	assert.Equal(t, "AI", memory.AIPrefix)
}

// TestMemoryVariables tests the MemoryVariables method
func TestChatMessageBufferMemory_MemoryVariables(t *testing.T) {
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	memory.MemoryKey = "custom_history"
	variables := memory.MemoryVariables()

	assert.Equal(t, []string{"custom_history"}, variables)
}

// TestLoadMemoryVariables_ReturnMessages tests loading when ReturnMessages is true
func TestChatMessageBufferMemory_LoadMemoryVariables_ReturnMessages(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	// Add some messages
	err := history.AddUserMessage(ctx, "Hello")
	require.NoError(t, err)
	err = history.AddAIMessage(ctx, "Hi there!")
	require.NoError(t, err)

	// Load memory variables
	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	assert.NoError(t, err)
	assert.Contains(t, vars, "history")

	messages, ok := vars["history"].([]schema.Message)
	assert.True(t, ok)
	assert.Len(t, messages, 2)
	assert.Equal(t, "Hello", messages[0].GetContent())
	assert.Equal(t, "Hi there!", messages[1].GetContent())
}

// TestLoadMemoryVariables_ReturnFormattedString tests loading when ReturnMessages is false
func TestChatMessageBufferMemory_LoadMemoryVariables_ReturnFormattedString(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)
	memory.ReturnMessages = false

	// Add some messages
	err := history.AddUserMessage(ctx, "Hello")
	require.NoError(t, err)
	err = history.AddAIMessage(ctx, "Hi there!")
	require.NoError(t, err)

	// Load memory variables
	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	assert.NoError(t, err)
	assert.Contains(t, vars, "history")

	formatted, ok := vars["history"].(string)
	assert.True(t, ok)
	assert.Contains(t, formatted, "Human: Hello")
	assert.Contains(t, formatted, "AI: Hi there!")
}

// TestLoadMemoryVariables_GetMessagesError tests error handling when GetMessages fails
func TestChatMessageBufferMemory_LoadMemoryVariables_GetMessagesError(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	history.getError = errors.New("get messages error")
	memory := NewChatMessageBufferMemory(history)

	_, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get messages from chat history")
}

// TestSaveContext tests saving context with default keys
func TestChatMessageBufferMemory_SaveContext(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.NoError(t, err)

	// Verify messages were added
	messages, err := history.GetMessages(ctx)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "Hello", messages[0].GetContent())
	assert.Equal(t, "Hi there!", messages[1].GetContent())
}

// TestSaveContext_CustomKeys tests saving context with custom keys
func TestChatMessageBufferMemory_SaveContext_CustomKeys(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)
	memory.InputKey = "query"
	memory.OutputKey = "response"

	inputs := map[string]any{"query": "Hello"}
	outputs := map[string]any{"response": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.NoError(t, err)

	// Verify messages were added
	messages, err := history.GetMessages(ctx)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
}

// TestSaveContext_AutoDetectKeys tests automatic key detection
func TestChatMessageBufferMemory_SaveContext_AutoDetectKeys(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)
	memory.InputKey = "" // Clear to trigger auto-detection
	memory.OutputKey = ""

	inputs := map[string]any{"question": "Hello"}
	outputs := map[string]any{"answer": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.NoError(t, err)

	// Verify messages were added
	messages, err := history.GetMessages(ctx)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
}

// TestSaveContext_MissingInputKey tests error when input key is missing
func TestChatMessageBufferMemory_SaveContext_MissingInputKey(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	inputs := map[string]any{"wrong_key": "Hello"}
	outputs := map[string]any{"output": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input key input not found in inputs")
}

// TestSaveContext_MissingOutputKey tests error when output key is missing
func TestChatMessageBufferMemory_SaveContext_MissingOutputKey(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"wrong_key": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output key output not found in outputs")
}

// TestSaveContext_NonStringInput tests error when input is not a string
func TestChatMessageBufferMemory_SaveContext_NonStringInput(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	inputs := map[string]any{"input": 123}
	outputs := map[string]any{"output": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input value must be a string")
}

// TestSaveContext_NonStringOutput tests error when output is not a string
func TestChatMessageBufferMemory_SaveContext_NonStringOutput(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": 456}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output value must be a string")
}

// TestSaveContext_AddUserMessageError tests error when adding user message fails
func TestChatMessageBufferMemory_SaveContext_AddUserMessageError(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	history.addError = errors.New("add user message error")
	memory := NewChatMessageBufferMemory(history)

	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add user message")
}

// TestSaveContext_AddAIMessageError tests error when adding AI message fails
func TestChatMessageBufferMemory_SaveContext_AddAIMessageError(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	// Add a failing mock for AddAIMessage specifically
	history.addMessageFunc = func(ctx context.Context, message schema.Message) error {
		if message.GetType() == schema.RoleAssistant {
			return errors.New("add AI message error")
		}
		return nil // Success for other message types
	}

	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add AI message")
}

// TestClear tests the Clear method
func TestChatMessageBufferMemory_Clear(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	// Add some messages first
	err := history.AddUserMessage(ctx, "Hello")
	require.NoError(t, err)

	// Clear the memory
	err = memory.Clear(ctx)
	assert.NoError(t, err)

	// Verify history is cleared
	messages, err := history.GetMessages(ctx)
	require.NoError(t, err)
	assert.Len(t, messages, 0)
}

// TestClear_HistoryError tests error handling when history Clear fails
func TestChatMessageBufferMemory_Clear_HistoryError(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	history.clearError = errors.New("clear error")
	memory := NewChatMessageBufferMemory(history)

	err := memory.Clear(ctx)
	assert.Error(t, err)
	assert.Equal(t, history.clearError, err)
}

// TestGetBufferString tests the getBufferString function
func TestGetBufferString(t *testing.T) {
	messages := []schema.Message{
		schema.NewHumanMessage("Hello"),
		schema.NewAIMessage("Hi there!"),
		schema.NewSystemMessage("System message"),
		schema.NewToolMessage("Tool result", "tool_call_123"),
		schema.NewHumanMessage("Unknown message"), // Using HumanMessage as generic fallback
	}

	result := getBufferString(messages, "Human", "AI")

	assert.Contains(t, result, "Human: Hello")
	assert.Contains(t, result, "AI: Hi there!")
	assert.Contains(t, result, "System: System message")
	assert.Contains(t, result, "Tool (tool_call_123): Tool result")
	assert.Contains(t, result, "Human: Unknown message")
}

// TestGetBufferString_Empty tests with empty message list
func TestGetBufferString_Empty(t *testing.T) {
	result := getBufferString([]schema.Message{}, "Human", "AI")
	assert.Equal(t, "", result)
}

// TestGetInputOutputKeys tests the getInputOutputKeys function
func TestGetInputOutputKeys(t *testing.T) {
	testCases := []struct {
		name           string
		inputs         map[string]any
		outputs        map[string]any
		expectedInput  string
		expectedOutput string
	}{
		{
			name:           "Empty maps",
			inputs:         map[string]any{},
			outputs:        map[string]any{},
			expectedInput:  "input",
			expectedOutput: "output",
		},
		{
			name:           "Standard keys",
			inputs:         map[string]any{"input": "test"},
			outputs:        map[string]any{"output": "test"},
			expectedInput:  "input",
			expectedOutput: "output",
		},
		{
			name:           "Alternative keys",
			inputs:         map[string]any{"question": "test"},
			outputs:        map[string]any{"answer": "test"},
			expectedInput:  "question",
			expectedOutput: "answer",
		},
		{
			name:           "Fallback to first key",
			inputs:         map[string]any{"custom_input": "test"},
			outputs:        map[string]any{"custom_output": "test"},
			expectedInput:  "custom_input",
			expectedOutput: "custom_output",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputKey, outputKey := getInputOutputKeys(tc.inputs, tc.outputs)
			assert.Equal(t, tc.expectedInput, inputKey)
			assert.Equal(t, tc.expectedOutput, outputKey)
		})
	}
}

// TestChatMessageBufferMemory_InterfaceCompliance tests that ChatMessageBufferMemory implements iface.Memory
func TestChatMessageBufferMemory_InterfaceCompliance(t *testing.T) {
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	// This will fail to compile if the interface is not implemented
	var _ iface.Memory = memory
}

// TestChatMessageBufferMemory_ConcurrentAccess tests concurrent access (basic smoke test)
func TestChatMessageBufferMemory_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	// This is a basic smoke test for concurrent access
	// In a real scenario, you'd want to use proper synchronization
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 10; i++ {
			inputs := map[string]any{"input": "Hello"}
			outputs := map[string]any{"output": "Hi!"}
			memory.SaveContext(ctx, inputs, outputs)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			memory.LoadMemoryVariables(ctx, map[string]any{})
		}
		done <- true
	}()

	<-done
	<-done
}

// BenchmarkChatMessageBufferMemory_SaveContext benchmarks SaveContext performance
func BenchmarkChatMessageBufferMemory_SaveContext(b *testing.B) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	inputs := map[string]any{"input": "Hello world"}
	outputs := map[string]any{"output": "Hi there!"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memory.SaveContext(ctx, inputs, outputs)
	}
}

// BenchmarkChatMessageBufferMemory_LoadMemoryVariables benchmarks LoadMemoryVariables performance
func BenchmarkChatMessageBufferMemory_LoadMemoryVariables(b *testing.B) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	// Pre-populate with some messages
	for i := 0; i < 100; i++ {
		history.AddUserMessage(ctx, "Message "+string(rune(i)))
		history.AddAIMessage(ctx, "Response "+string(rune(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memory.LoadMemoryVariables(ctx, map[string]any{})
	}
}

// BenchmarkGetBufferString benchmarks the getBufferString function
func BenchmarkGetBufferString(b *testing.B) {
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

// TestGetBufferString_CustomPrefixes tests with custom prefixes
func TestGetBufferString_CustomPrefixes(t *testing.T) {
	messages := []schema.Message{
		schema.NewHumanMessage("Hello"),
		schema.NewAIMessage("Hi!"),
	}

	result := getBufferString(messages, "User", "Bot")

	lines := strings.Split(strings.TrimSpace(result), "\n")
	assert.Len(t, lines, 2)
	assert.Contains(t, lines[0], "User: Hello")
	assert.Contains(t, lines[1], "Bot: Hi!")
}

// TestGetBufferString_SystemMessage tests system message formatting
func TestGetBufferString_SystemMessage(t *testing.T) {
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant"),
	}

	result := getBufferString(messages, "Human", "AI")
	assert.Contains(t, result, "System: You are a helpful assistant")
}

// TestGetBufferString_ToolMessage tests tool message formatting
func TestGetBufferString_ToolMessage(t *testing.T) {
	messages := []schema.Message{
		schema.NewToolMessage("Tool result", "call_123"),
	}

	result := getBufferString(messages, "Human", "AI")
	assert.Contains(t, result, "Tool (call_123): Tool result")
}

// TestGetBufferString_CustomMessage tests custom message formatting
func TestGetBufferString_CustomMessage(t *testing.T) {
	// Create a custom message by using HumanMessage with different prefix
	messages := []schema.Message{
		schema.NewHumanMessage("Custom content"),
	}

	result := getBufferString(messages, "Custom", "AI")
	assert.Contains(t, result, "Custom: Custom content")
}
