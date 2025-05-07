// Package memory provides interfaces and implementations for managing conversation history.
package memory

import (
	"context"
	"fmt"
)

// ChatMessageBufferMemory is a simple memory implementation that stores all messages in a buffer.
type ChatMessageBufferMemory struct {
	ChatHistory    ChatMessageHistory
	ReturnMessages bool   // Whether to return messages directly or as formatted string
	MemoryKey      string // Key used for storing the memory in prompt variables
	InputKey       string // Key for input in SaveContext
	OutputKey      string // Key for output in SaveContext
	HumanPrefix    string // Prefix for human messages when formatted
	AIPrefix       string // Prefix for AI messages when formatted
}

// NewChatMessageBufferMemory creates a new buffer memory with default settings.
func NewChatMessageBufferMemory(history ChatMessageHistory) *ChatMessageBufferMemory {
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
	buffer := GetBufferString(messages, m.HumanPrefix, m.AIPrefix)
	return map[string]any{m.MemoryKey: buffer}, nil
}

// SaveContext saves a new interaction to memory.
func (m *ChatMessageBufferMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	inputKey := m.InputKey
	outputKey := m.OutputKey
	var err error

	if inputKey == "" || outputKey == "" {
		inputKey, outputKey, err = GetInputOutputKeys(inputs, outputs)
		if err != nil {
			return fmt.Errorf("failed to determine input/output keys: %w", err)
		}
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

// Clear empties the chat history.
func (m *ChatMessageBufferMemory) Clear(ctx context.Context) error {
	return m.ChatHistory.Clear(ctx)
}
