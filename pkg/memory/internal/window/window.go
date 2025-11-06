// Package window provides window-based memory implementations.
// It contains implementations that maintain a fixed-size window of recent interactions.
package window

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// ConversationBufferWindowMemory remembers a fixed number of the most recent interactions.
// It loads the history into a single variable ("history" by default).
type ConversationBufferWindowMemory struct {
	ChatHistory    iface.ChatMessageHistory // Underlying storage for messages
	K              int                      // Number of messages to keep in the window
	MemoryKey      string                   // Key name for the history variable in prompts
	InputKey       string                   // Key name for the user input variable (optional, used in SaveContext)
	OutputKey      string                   // Key name for the AI output variable (optional, used in SaveContext)
	HumanPrefix    string                   // Prefix for human messages in the buffer string
	AiPrefix       string                   // Prefix for AI messages in the buffer string
	ReturnMessages bool                     // If true, LoadMemoryVariables returns []schema.Message, otherwise a formatted string
}

// NewConversationBufferWindowMemory creates a new ConversationBufferWindowMemory.
func NewConversationBufferWindowMemory(history iface.ChatMessageHistory, k int, memoryKey string, returnMessages bool) *ConversationBufferWindowMemory {
	key := memoryKey
	if key == "" {
		key = "history" // Default memory key
	}
	if k <= 0 {
		k = 5 // Default window size
	}
	return &ConversationBufferWindowMemory{
		ChatHistory:    history,
		K:              k,
		MemoryKey:      key,
		HumanPrefix:    "Human",
		AiPrefix:       "AI",
		ReturnMessages: returnMessages,
	}
}

// MemoryVariables returns the key used for the conversation history.
func (m *ConversationBufferWindowMemory) MemoryVariables() []string {
	return []string{m.MemoryKey}
}

// LoadMemoryVariables retrieves the conversation history, pruned to the window size (K).
// It returns either a formatted string or a slice of messages based on ReturnMessages.
func (m *ConversationBufferWindowMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	messages, err := m.ChatHistory.GetMessages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages from chat history: %w", err)
	}

	// Prune messages to keep only the last K
	if len(messages) > m.K {
		messages = messages[len(messages)-m.K:]
	}

	var memoryValue any
	if m.ReturnMessages {
		memoryValue = messages
	} else {
		memoryValue = getBufferString(messages, m.HumanPrefix, m.AiPrefix)
	}

	return map[string]any{m.MemoryKey: memoryValue}, nil
}

// SaveContext adds the latest user input and AI output to the chat history.
// Pruning happens during loading, not saving.
func (m *ConversationBufferWindowMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	if m.ChatHistory == nil {
		return fmt.Errorf("chat history is nil")
	}

	inputKey := m.InputKey
	outputKey := m.OutputKey

	// Determine input/output keys if not explicitly set
	if inputKey == "" || outputKey == "" {
		detectedInputKey, detectedOutputKey := getInputOutputKeys(inputs, outputs)
		if inputKey == "" {
			inputKey = detectedInputKey
		}
		if outputKey == "" {
			outputKey = detectedOutputKey
		}
	}

	inputVal, inputOk := inputs[inputKey]
	outputVal, outputOk := outputs[outputKey]

	if !inputOk {
		return fmt.Errorf("input key %s not found in inputs map", inputKey)
	}
	if !outputOk {
		return fmt.Errorf("output key %s not found in outputs map", outputKey)
	}

	inputStr, inputStrOk := inputVal.(string)
	outputStr, outputStrOk := outputVal.(string)

	if !inputStrOk {
		return fmt.Errorf("input value for key %s is not a string (type: %T)", inputKey, inputVal)
	}
	if !outputStrOk {
		return fmt.Errorf("output value for key %s is not a string (type: %T)", outputKey, outputVal)
	}

	err := m.ChatHistory.AddUserMessage(ctx, inputStr)
	if err != nil {
		return fmt.Errorf("failed to add user message to chat history: %w", err)
	}
	err = m.ChatHistory.AddAIMessage(ctx, outputStr)
	if err != nil {
		return fmt.Errorf("failed to add AI message to chat history: %w", err)
	}

	return nil
}

// Clear removes all messages from the underlying chat history.
func (m *ConversationBufferWindowMemory) Clear(ctx context.Context) error {
	return m.ChatHistory.Clear(ctx)
}

// getInputOutputKeys determines the input and output keys from the given maps.
func getInputOutputKeys(inputs map[string]any, outputs map[string]any) (string, string) {
	if len(inputs) == 0 || len(outputs) == 0 {
		return "input", "output"
	}

	// Common input/output key names
	possibleInputKeys := []string{"input", "query", "question", "human_input", "user_input"}
	possibleOutputKeys := []string{"output", "result", "answer", "ai_output", "response"}

	// Try to find known input key
	var inputKey string
	for _, key := range possibleInputKeys {
		if _, ok := inputs[key]; ok {
			inputKey = key
			break
		}
	}

	// If no known input key, use the first key
	if inputKey == "" {
		for k := range inputs {
			inputKey = k
			break
		}
	}

	// Try to find known output key
	var outputKey string
	for _, key := range possibleOutputKeys {
		if _, ok := outputs[key]; ok {
			outputKey = key
			break
		}
	}

	// If no known output key, use the first key
	if outputKey == "" {
		for k := range outputs {
			outputKey = k
			break
		}
	}

	return inputKey, outputKey
}

// getBufferString formats messages into a text buffer with human/AI prefixes.
func getBufferString(messages []schema.Message, humanPrefix, aiPrefix string) string {
	var buffer strings.Builder

	for _, msg := range messages {
		switch msg.GetType() {
		case schema.RoleHuman:
			buffer.WriteString(fmt.Sprintf("%s: %s\n", humanPrefix, msg.GetContent()))
		case schema.RoleAssistant:
			buffer.WriteString(fmt.Sprintf("%s: %s\n", aiPrefix, msg.GetContent()))
		case schema.RoleSystem:
			buffer.WriteString(fmt.Sprintf("System: %s\n", msg.GetContent()))
		case schema.RoleTool:
			toolMsg, ok := msg.(*schema.ToolMessage)
			if ok {
				buffer.WriteString(fmt.Sprintf("Tool (%s): %s\n", toolMsg.ToolCallID, msg.GetContent()))
			} else {
				buffer.WriteString(fmt.Sprintf("Tool: %s\n", msg.GetContent()))
			}
		default:
			buffer.WriteString(fmt.Sprintf("%s: %s\n", msg.GetType(), msg.GetContent()))
		}
	}

	return buffer.String()
}

// Ensure ConversationBufferWindowMemory implements the interface.
var _ iface.Memory = (*ConversationBufferWindowMemory)(nil)
