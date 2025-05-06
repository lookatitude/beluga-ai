// Package memory provides interfaces and implementations for managing conversation history.
package memory

import (
	"context"
	"fmt"
)

// ConversationBufferMemory remembers the entire conversation history.
// It loads the history into a single variable ("history" by default).
type ConversationBufferMemory struct {
	ChatHistory    ChatMessageHistory // Underlying storage for messages
	MemoryKey      string             // Key name for the history variable in prompts
	InputKey       string             // Key name for the user input variable (optional, used in SaveContext)
	OutputKey      string             // Key name for the AI output variable (optional, used in SaveContext)
	HumanPrefix    string             // Prefix for human messages in the buffer string
	AiPrefix       string             // Prefix for AI messages in the buffer string
	ReturnMessages bool               // If true, LoadMemoryVariables returns []schema.Message, otherwise a formatted string
}

// NewConversationBufferMemory creates a new ConversationBufferMemory.
func NewConversationBufferMemory(history ChatMessageHistory, memoryKey string, returnMessages bool) *ConversationBufferMemory {
	key := memoryKey
	if key == "" {
		key = "history" // Default memory key
	}
	return &ConversationBufferMemory{
		ChatHistory:    history,
		MemoryKey:      key,
		HumanPrefix:    "Human",
		AiPrefix:       "AI",
		ReturnMessages: returnMessages,
	}
}

// MemoryVariables returns the key used for the conversation history.
func (m *ConversationBufferMemory) MemoryVariables() []string {
	return []string{m.MemoryKey}
}

// LoadMemoryVariables retrieves the conversation history.
// It returns either a formatted string or a slice of messages based on ReturnMessages.
func (m *ConversationBufferMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	messages, err := m.ChatHistory.GetMessages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages from chat history: %w", err)
	}

	var memoryValue any
	if m.ReturnMessages {
		memoryValue = messages
	} else {
		memoryValue = GetBufferString(messages, m.HumanPrefix, m.AiPrefix)
	}

	return map[string]any{m.MemoryKey: memoryValue}, nil
}

// SaveContext adds the latest user input and AI output to the chat history.
func (m *ConversationBufferMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	inputKey := m.InputKey
	outputKey := m.OutputKey
	var err error

	// Determine input/output keys if not explicitly set
	if inputKey == "" || outputKey == "" {
		inputKey, outputKey, err = GetInputOutputKeys(inputs, outputs)
		if err != nil {
			return fmt.Errorf("failed to determine input/output keys for saving context: %w", err)
		}
	}

	inputVal, inputOk := inputs[inputKey]
	outputVal, outputOk := outputs[outputKey]

	if !inputOk {
		return fmt.Errorf("input key 	%s	 not found in inputs map", inputKey)
	}
	if !outputOk {
		return fmt.Errorf("output key 	%s	 not found in outputs map", outputKey)
	}

	inputStr, inputStrOk := inputVal.(string)
	outputStr, outputStrOk := outputVal.(string)

	if !inputStrOk {
		return fmt.Errorf("input value for key 	%s	 is not a string (type: %T)", inputKey, inputVal)
	}
	if !outputStrOk {
		return fmt.Errorf("output value for key 	%s	 is not a string (type: %T)", outputKey, outputVal)
	}

	err = m.ChatHistory.AddUserMessage(ctx, inputStr)
	if err != nil {
		return fmt.Errorf("failed to add user message to history: %w", err)
	}
	err = m.ChatHistory.AddAIMessage(ctx, outputStr)
	if err != nil {
		return fmt.Errorf("failed to add AI message to history: %w", err)
	}

	return nil
}

// Clear removes all messages from the underlying chat history.
func (m *ConversationBufferMemory) Clear(ctx context.Context) error {
	return m.ChatHistory.Clear(ctx)
}

// Ensure ConversationBufferMemory implements the interface.
var _ BaseMemory = (*ConversationBufferMemory)(nil)
