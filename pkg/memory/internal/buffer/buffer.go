// Package buffer provides buffer memory implementations.
// It contains the concrete implementations for buffer-based memory types.
package buffer

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// ChatMessageBufferMemory is a simple memory implementation that stores all messages in a buffer.
type ChatMessageBufferMemory struct {
	ChatHistory    iface.ChatMessageHistory
	ReturnMessages bool   // Whether to return messages directly or as formatted string
	MemoryKey      string // Key used for storing the memory in prompt variables
	InputKey       string // Key for input in SaveContext
	OutputKey      string // Key for output in SaveContext
	HumanPrefix    string // Prefix for human messages when formatted
	AIPrefix       string // Prefix for AI messages when formatted
}

// NewChatMessageBufferMemory creates a new buffer memory with default settings.
func NewChatMessageBufferMemory(history iface.ChatMessageHistory) *ChatMessageBufferMemory {
	return &ChatMessageBufferMemory{
		ChatHistory:    history,
		ReturnMessages: true,
		MemoryKey:      "history",
		InputKey:       "input",
		OutputKey:      "output",
		HumanPrefix:    "Human",
		AIPrefix:       "AI",
	}
}

// MemoryVariables returns the variables exposed by this memory implementation.
func (m *ChatMessageBufferMemory) MemoryVariables() []string {
	return []string{m.MemoryKey}
}

// LoadMemoryVariables loads messages from the chat history.
func (m *ChatMessageBufferMemory) LoadMemoryVariables(ctx context.Context, _ map[string]any) (map[string]any, error) {
	messages, err := m.ChatHistory.GetMessages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages from chat history: %w", err)
	}

	if m.ReturnMessages {
		return map[string]any{m.MemoryKey: messages}, nil
	}

	// Format messages as a string
	buffer := getBufferString(messages, m.HumanPrefix, m.AIPrefix)
	return map[string]any{m.MemoryKey: buffer}, nil
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

// SaveContext saves a new interaction to memory.
func (m *ChatMessageBufferMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	inputKey := m.InputKey
	outputKey := m.OutputKey

	if inputKey == "" || outputKey == "" {
		inputKey, outputKey = getInputOutputKeys(inputs, outputs)
	}

	inputVal, inputOk := inputs[inputKey]
	outputVal, outputOk := outputs[outputKey]

	if !inputOk {
		return fmt.Errorf("input key %s not found in inputs", inputKey)
	}
	if !outputOk {
		return fmt.Errorf("output key %s not found in outputs", outputKey)
	}

	inputStr, inputIsStr := inputVal.(string)
	if !inputIsStr {
		return fmt.Errorf("input value must be a string, got %T", inputVal)
	}

	outputStr, outputIsStr := outputVal.(string)
	if !outputIsStr {
		return fmt.Errorf("output value must be a string, got %T", outputVal)
	}

	if err := m.ChatHistory.AddUserMessage(ctx, inputStr); err != nil {
		return fmt.Errorf("failed to add user message: %w", err)
	}

	if err := m.ChatHistory.AddAIMessage(ctx, outputStr); err != nil {
		return fmt.Errorf("failed to add AI message: %w", err)
	}

	return nil
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

// Clear empties the chat history.
func (m *ChatMessageBufferMemory) Clear(ctx context.Context) error {
	return m.ChatHistory.Clear(ctx)
}

// Ensure ChatMessageBufferMemory implements the interface.
var _ iface.Memory = (*ChatMessageBufferMemory)(nil)
