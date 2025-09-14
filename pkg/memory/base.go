// Package memory provides interfaces and implementations for managing conversation history.
package memory

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Memory defines the interface for all memory implementations.
type Memory interface {
	// MemoryVariables returns the list of variable names that the memory makes available.
	MemoryVariables() []string

	// LoadMemoryVariables loads memory variables given the context and input values.
	LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error)

	// SaveContext saves the current context and new inputs/outputs to memory.
	SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error

	// Clear clears the memory contents.
	Clear(ctx context.Context) error
}

// BaseMemory is an alias for Memory to maintain compatibility.
type BaseMemory = Memory

// ChatMessageHistory defines the interface for storing and retrieving message history.
type ChatMessageHistory interface {
	// AddMessage adds a message to the history.
	AddMessage(ctx context.Context, message schema.Message) error

	// AddUserMessage adds a human message to the history.
	AddUserMessage(ctx context.Context, content string) error

	// AddAIMessage adds an AI message to the history.
	AddAIMessage(ctx context.Context, content string) error

	// GetMessages returns all messages in the history.
	GetMessages(ctx context.Context) ([]schema.Message, error)

	// Clear removes all messages from the history.
	Clear(ctx context.Context) error
}

// GetInputOutputKeys determines the input and output keys from the given maps.
func GetInputOutputKeys(inputs map[string]any, outputs map[string]any) (string, string, error) {
	if len(inputs) == 0 {
		return "", "", errors.New("inputs map is empty")
	}
	if len(outputs) == 0 {
		return "", "", errors.New("outputs map is empty")
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

	return inputKey, outputKey, nil
}

// GetBufferString formats messages into a text buffer with human/AI prefixes.
func GetBufferString(messages []schema.Message, humanPrefix, aiPrefix string) string {
	var buffer strings.Builder

	for _, msg := range messages {
		switch msg.GetType() {
		case schema.MessageTypeHuman:
			buffer.WriteString(fmt.Sprintf("%s: %s\n", humanPrefix, msg.GetContent()))
		case schema.MessageTypeAI:
			buffer.WriteString(fmt.Sprintf("%s: %s\n", aiPrefix, msg.GetContent()))
		case schema.MessageTypeSystem:
			buffer.WriteString(fmt.Sprintf("System: %s\n", msg.GetContent()))
		case schema.MessageTypeTool:
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
