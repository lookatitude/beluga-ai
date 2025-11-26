// Package window provides comprehensive tests for window-based memory implementations.
package window

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockChatMessageHistory is a mock implementation for testing.
type MockChatMessageHistory struct {
	addError   error
	getError   error
	clearError error
	messages   []schema.Message
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

// Ensure MockChatMessageHistory implements the interface.
var _ iface.ChatMessageHistory = (*MockChatMessageHistory)(nil)

// TestNewConversationBufferWindowMemory tests the constructor.
func TestNewConversationBufferWindowMemory(t *testing.T) {
	history := NewMockChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, 5, "test_history", false)

	assert.NotNil(t, memory)
	assert.Equal(t, history, memory.ChatHistory)
	assert.Equal(t, 5, memory.K)
	assert.Equal(t, "test_history", memory.MemoryKey)
	assert.False(t, memory.ReturnMessages)
	assert.Equal(t, "Human", memory.HumanPrefix)
	assert.Equal(t, "AI", memory.AiPrefix)
}

// TestNewConversationBufferWindowMemory_Defaults tests default values.
func TestNewConversationBufferWindowMemory_Defaults(t *testing.T) {
	history := NewMockChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, 0, "", true)

	assert.Equal(t, 5, memory.K)                 // Default window size
	assert.Equal(t, "history", memory.MemoryKey) // Default memory key
	assert.True(t, memory.ReturnMessages)
}

// TestConversationBufferWindowMemory_MemoryVariables tests the MemoryVariables method.
func TestConversationBufferWindowMemory_MemoryVariables(t *testing.T) {
	history := NewMockChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, 5, "custom_history", false)

	variables := memory.MemoryVariables()
	assert.Equal(t, []string{"custom_history"}, variables)
}

// TestConversationBufferWindowMemory_LoadMemoryVariables_ReturnMessages tests loading with ReturnMessages=true.
func TestConversationBufferWindowMemory_LoadMemoryVariables_ReturnMessages(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, 3, "history", true)

	// Add more messages than the window size
	for i := 0; i < 5; i++ {
		err := history.AddUserMessage(ctx, "User message "+string(rune(i+'0')))
		require.NoError(t, err)
		err = history.AddAIMessage(ctx, "AI message "+string(rune(i+'0')))
		require.NoError(t, err)
	}

	// Load memory variables
	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, vars, "history")

	messages, ok := vars["history"].([]schema.Message)
	assert.True(t, ok)
	assert.Len(t, messages, 3) // Should only return last 3 messages due to window size
	// With 10 messages (5 user + 5 AI) and window of 3, should return last 3 messages
	// Messages are: User0, AI0, User1, AI1, User2, AI2, User3, AI3, User4, AI4
	// Last 3: AI3, User4, AI4
	assert.Equal(t, "AI message 3", messages[0].GetContent())
	assert.Equal(t, "User message 4", messages[1].GetContent())
	assert.Equal(t, "AI message 4", messages[2].GetContent())
}

// TestConversationBufferWindowMemory_LoadMemoryVariables_ReturnFormattedString tests loading with ReturnMessages=false.
func TestConversationBufferWindowMemory_LoadMemoryVariables_ReturnFormattedString(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, 2, "history", false)

	// Add messages
	err := history.AddUserMessage(ctx, "Hello")
	require.NoError(t, err)
	err = history.AddAIMessage(ctx, "Hi!")
	require.NoError(t, err)
	err = history.AddUserMessage(ctx, "How are you?")
	require.NoError(t, err)
	err = history.AddAIMessage(ctx, "I'm fine!")
	require.NoError(t, err)

	// Load memory variables
	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, vars, "history")

	formatted, ok := vars["history"].(string)
	assert.True(t, ok)
	assert.Contains(t, formatted, "How are you?")
	assert.Contains(t, formatted, "I'm fine!")
	// Should not contain the first two messages due to window size
	assert.NotContains(t, formatted, "Hello")
}

// TestConversationBufferWindowMemory_LoadMemoryVariables_FewerThanWindow tests when there are fewer messages than window size.
func TestConversationBufferWindowMemory_LoadMemoryVariables_FewerThanWindow(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, 5, "history", true)

	// Add fewer messages than window size
	err := history.AddUserMessage(ctx, "Hello")
	require.NoError(t, err)
	err = history.AddAIMessage(ctx, "Hi!")
	require.NoError(t, err)

	// Load memory variables
	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	require.NoError(t, err)

	messages, ok := vars["history"].([]schema.Message)
	assert.True(t, ok)
	assert.Len(t, messages, 2) // Should return all messages since fewer than window size
}

// TestConversationBufferWindowMemory_LoadMemoryVariables_GetMessagesError tests error handling.
func TestConversationBufferWindowMemory_LoadMemoryVariables_GetMessagesError(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	history.getError = errors.New("get messages error")
	memory := NewConversationBufferWindowMemory(history, 3, "history", true)

	_, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get messages from chat history")
}

// TestConversationBufferWindowMemory_SaveContext tests saving context.
func TestConversationBufferWindowMemory_SaveContext(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, 5, "history", true)

	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	require.NoError(t, err)

	// Verify messages were added
	messages, err := history.GetMessages(ctx)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "Hello", messages[0].GetContent())
	assert.Equal(t, "Hi there!", messages[1].GetContent())
}

// TestConversationBufferWindowMemory_SaveContext_CustomKeys tests custom keys.
func TestConversationBufferWindowMemory_SaveContext_CustomKeys(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, 5, "history", true)
	memory.InputKey = "question"
	memory.OutputKey = "answer"

	inputs := map[string]any{"question": "Hello"}
	outputs := map[string]any{"answer": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	require.NoError(t, err)

	messages, err := history.GetMessages(ctx)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
}

// TestConversationBufferWindowMemory_SaveContext_AutoDetectKeys tests automatic key detection.
func TestConversationBufferWindowMemory_SaveContext_AutoDetectKeys(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, 5, "history", true)
	// Clear keys to trigger auto-detection
	memory.InputKey = ""
	memory.OutputKey = ""

	inputs := map[string]any{"query": "Hello"}
	outputs := map[string]any{"response": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	require.NoError(t, err)

	messages, err := history.GetMessages(ctx)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
}

// TestConversationBufferWindowMemory_SaveContext_ErrorHandling tests various error conditions.
func TestConversationBufferWindowMemory_SaveContext_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name          string
		setupMemory   func(*ConversationBufferWindowMemory)
		inputs        map[string]any
		outputs       map[string]any
		expectedError string
	}{
		{
			name: "Missing input key",
			setupMemory: func(m *ConversationBufferWindowMemory) {
				m.InputKey = "input"
			},
			inputs:        map[string]any{"wrong_key": "Hello"},
			outputs:       map[string]any{"output": "Hi!"},
			expectedError: "input key input not found in inputs map",
		},
		{
			name: "Missing output key",
			setupMemory: func(m *ConversationBufferWindowMemory) {
				m.OutputKey = "output"
			},
			inputs:        map[string]any{"input": "Hello"},
			outputs:       map[string]any{"wrong_key": "Hi!"},
			expectedError: "output key output not found in outputs map",
		},
		{
			name: "Non-string input",
			setupMemory: func(m *ConversationBufferWindowMemory) {
				m.InputKey = "input"
				m.OutputKey = "output"
			},
			inputs:        map[string]any{"input": 123},
			outputs:       map[string]any{"output": "Hi!"},
			expectedError: "input value for key input is not a string",
		},
		{
			name: "Non-string output",
			setupMemory: func(m *ConversationBufferWindowMemory) {
				m.InputKey = "input"
				m.OutputKey = "output"
			},
			inputs:        map[string]any{"input": "Hello"},
			outputs:       map[string]any{"output": 456},
			expectedError: "output value for key output is not a string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			history := NewMockChatMessageHistory()
			memory := NewConversationBufferWindowMemory(history, 5, "history", true)
			tc.setupMemory(memory)

			err := memory.SaveContext(ctx, tc.inputs, tc.outputs)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

// TestConversationBufferWindowMemory_SaveContext_AddMessageErrors tests errors from chat history.
func TestConversationBufferWindowMemory_SaveContext_AddMessageErrors(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, 5, "history", true)

	// Test AddUserMessage error
	history.addError = errors.New("add user message error")
	err := memory.SaveContext(ctx, map[string]any{"input": "Hello"}, map[string]any{"output": "Hi!"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add user message")
}

// TestConversationBufferWindowMemory_Clear tests the Clear method.
func TestConversationBufferWindowMemory_Clear(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, 5, "history", true)

	// Add some messages
	err := history.AddUserMessage(ctx, "Hello")
	require.NoError(t, err)

	// Clear
	err = memory.Clear(ctx)
	require.NoError(t, err)

	// Verify cleared
	messages, err := history.GetMessages(ctx)
	require.NoError(t, err)
	assert.Empty(t, messages)
}

// TestConversationBufferWindowMemory_Clear_Error tests error handling in Clear.
func TestConversationBufferWindowMemory_Clear_Error(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	history.clearError = errors.New("clear error")
	memory := NewConversationBufferWindowMemory(history, 5, "history", true)

	err := memory.Clear(ctx)
	require.Error(t, err)
	assert.Equal(t, history.clearError, err)
}

// TestConversationWindowBufferMemory tests the ConversationWindowBufferMemory implementation.
func TestConversationWindowBufferMemory(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewConversationWindowBufferMemory(history, 3)

	assert.NotNil(t, memory)
	assert.Equal(t, 3, memory.K)
	assert.Equal(t, history, memory.ChatHistory)
	assert.True(t, memory.ReturnMessages) // Default from ChatMessageBufferMemory

	// Add messages (6 messages = 3 interactions)
	for i := 0; i < 6; i++ {
		if i%2 == 0 {
			err := history.AddUserMessage(ctx, "User "+string(rune(i/2+'0')))
			require.NoError(t, err)
		} else {
			err := history.AddAIMessage(ctx, "AI "+string(rune(i/2+'0')))
			require.NoError(t, err)
		}
	}

	// Load memory - should return last 6 messages (3 interactions * 2 messages each)
	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, vars, "history")

	messages, ok := vars["history"].([]schema.Message)
	assert.True(t, ok)
	assert.Len(t, messages, 6) // Should return all 6 messages since window size is 3 interactions
}

// TestConversationWindowBufferMemory_LoadMemoryVariables_ReturnFormattedString tests formatted string return.
func TestConversationWindowBufferMemory_LoadMemoryVariables_ReturnFormattedString(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewConversationWindowBufferMemory(history, 2)
	memory.ReturnMessages = false

	// Add messages
	err := history.AddUserMessage(ctx, "Hello")
	require.NoError(t, err)
	err = history.AddAIMessage(ctx, "Hi!")
	require.NoError(t, err)
	err = history.AddUserMessage(ctx, "How are you?")
	require.NoError(t, err)
	err = history.AddAIMessage(ctx, "Fine!")
	require.NoError(t, err)

	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	require.NoError(t, err)

	formatted, ok := vars["history"].(string)
	assert.True(t, ok)
	assert.Contains(t, formatted, "Hello")
	assert.Contains(t, formatted, "Hi!")
	assert.Contains(t, formatted, "How are you?")
	assert.Contains(t, formatted, "Fine!")
}

// TestConversationWindowBufferMemory_DefaultK tests default window size.
func TestConversationWindowBufferMemory_DefaultK(t *testing.T) {
	history := NewMockChatMessageHistory()
	memory := NewConversationWindowBufferMemory(history, 0)

	assert.Equal(t, 5, memory.K) // Should default to 5
}

// TestGetBufferString tests the getBufferString function.
func TestGetBufferString_Window(t *testing.T) {
	messages := []schema.Message{
		schema.NewHumanMessage("Hello"),
		schema.NewAIMessage("Hi!"),
		schema.NewSystemMessage("System message"),
	}

	result := getBufferString(messages, "User", "Bot")

	assert.Contains(t, result, "User: Hello")
	assert.Contains(t, result, "Bot: Hi!")
	assert.Contains(t, result, "System: System message")
}

// TestGetInputOutputKeys tests the getInputOutputKeys function.
func TestGetInputOutputKeys_Window(t *testing.T) {
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputKey, outputKey := getInputOutputKeys(tc.inputs, tc.outputs)
			assert.Equal(t, tc.expectedInput, inputKey)
			assert.Equal(t, tc.expectedOutput, outputKey)
		})
	}
}

// TestInterfaceCompliance tests that both implementations comply with the Memory interface.
func TestInterfaceCompliance(t *testing.T) {
	history := NewMockChatMessageHistory()

	// Test ConversationBufferWindowMemory
	windowMemory := NewConversationBufferWindowMemory(history, 5, "history", true)
	var _ iface.Memory = windowMemory

	// Test ConversationWindowBufferMemory
	windowBufferMemory := NewConversationWindowBufferMemory(history, 3)
	var _ iface.Memory = windowBufferMemory
}

// BenchmarkConversationBufferWindowMemory_SaveContext benchmarks SaveContext performance.
func BenchmarkConversationBufferWindowMemory_SaveContext(b *testing.B) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, 10, "history", true)

	inputs := map[string]any{"input": "Hello world"}
	outputs := map[string]any{"output": "Hi there!"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = memory.SaveContext(ctx, inputs, outputs)
	}
}

// BenchmarkConversationBufferWindowMemory_LoadMemoryVariables benchmarks LoadMemoryVariables performance.
func BenchmarkConversationBufferWindowMemory_LoadMemoryVariables(b *testing.B) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, 10, "history", true)

	// Pre-populate with messages
	for i := 0; i < 100; i++ {
		_ = history.AddUserMessage(ctx, "Message "+string(rune(i)))
		_ = history.AddAIMessage(ctx, "Response "+string(rune(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = memory.LoadMemoryVariables(ctx, map[string]any{})
	}
}

// BenchmarkGetBufferString benchmarks the getBufferString function.
func BenchmarkGetBufferString_Window(b *testing.B) {
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

// TestConversationBufferWindowMemory_WindowPruning tests that window pruning works correctly.
func TestConversationBufferWindowMemory_WindowPruning(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, 2, "history", true)

	// Add 6 messages (3 interactions)
	for i := 0; i < 6; i++ {
		if i%2 == 0 {
			err := history.AddUserMessage(ctx, "User "+string(rune(i/2+'0')))
			require.NoError(t, err)
		} else {
			err := history.AddAIMessage(ctx, "AI "+string(rune(i/2+'0')))
			require.NoError(t, err)
		}
	}

	// Load memory - should return only last 2 messages due to window size
	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	require.NoError(t, err)

	messages, ok := vars["history"].([]schema.Message)
	assert.True(t, ok)
	assert.Len(t, messages, 2) // Window size of 2
	assert.Equal(t, "User 2", messages[0].GetContent())
	assert.Equal(t, "AI 2", messages[1].GetContent())
}

// TestConversationWindowBufferMemory_WindowPruning tests window pruning for ConversationWindowBufferMemory.
func TestConversationWindowBufferMemory_WindowPruning(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewConversationWindowBufferMemory(history, 2) // Keep last 2 interactions (4 messages)

	// Add 8 messages (4 interactions)
	for i := 0; i < 8; i++ {
		if i%2 == 0 {
			err := history.AddUserMessage(ctx, "User "+string(rune(i/2+'0')))
			require.NoError(t, err)
		} else {
			err := history.AddAIMessage(ctx, "AI "+string(rune(i/2+'0')))
			require.NoError(t, err)
		}
	}

	// Load memory - should return last 4 messages (2 interactions * 2 messages each)
	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	require.NoError(t, err)

	messages, ok := vars["history"].([]schema.Message)
	assert.True(t, ok)
	assert.Len(t, messages, 4) // 2 interactions * 2 messages each
	assert.Equal(t, "User 2", messages[0].GetContent())
	assert.Equal(t, "AI 2", messages[1].GetContent())
	assert.Equal(t, "User 3", messages[2].GetContent())
	assert.Equal(t, "AI 3", messages[3].GetContent())
}

// TestWindowMemory_CustomPrefixes tests custom message prefixes.
func TestWindowMemory_CustomPrefixes(t *testing.T) {
	ctx := context.Background()
	history := NewMockChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, 5, "history", false)
	memory.HumanPrefix = "User"
	memory.AiPrefix = "Assistant"

	err := history.AddUserMessage(ctx, "Hello")
	require.NoError(t, err)
	err = history.AddAIMessage(ctx, "Hi!")
	require.NoError(t, err)

	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	require.NoError(t, err)

	formatted, ok := vars["history"].(string)
	assert.True(t, ok)
	assert.Contains(t, formatted, "User: Hello")
	assert.Contains(t, formatted, "Assistant: Hi!")
}
