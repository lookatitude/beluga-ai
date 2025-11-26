// Package summary provides summary-based memory implementations.
// It contains implementations that condense conversations using LLMs.
package summary

import (
	"context"
	"errors"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	promptsiface "github.com/lookatitude/beluga-ai/pkg/prompts/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// ConversationSummaryBufferMemory combines buffer memory with summarization.
// It keeps a buffer of recent interactions and summarizes older ones using an LLM.
type ConversationSummaryBufferMemory struct {
	ChatHistory         iface.ChatMessageHistory
	LLM                 core.Runnable
	SummaryPrompt       promptsiface.Template
	MemoryKey           string
	InputKey            string
	OutputKey           string
	HumanPrefix         string
	AiPrefix            string
	movingSummaryBuffer string
	MaxTokenLimit       int
}

// NewConversationSummaryBufferMemory creates a new ConversationSummaryBufferMemory.
func NewConversationSummaryBufferMemory(history iface.ChatMessageHistory, llm core.Runnable, memoryKey string, maxTokenLimit int) *ConversationSummaryBufferMemory {
	key := memoryKey
	if key == "" {
		key = "history"
	}
	// Handle potential error from NewStringPromptTemplate
	summaryPrompt := DefaultSummaryPrompt
	if summaryPrompt == nil {
		panic("Failed to parse DefaultSummaryPrompt")
	}
	return &ConversationSummaryBufferMemory{
		ChatHistory:   history,
		LLM:           llm,
		SummaryPrompt: summaryPrompt,
		MemoryKey:     key,
		HumanPrefix:   "Human",
		AiPrefix:      "AI",
		MaxTokenLimit: maxTokenLimit,
	}
}

// MemoryVariables returns the key used for the conversation history/summary.
func (m *ConversationSummaryBufferMemory) MemoryVariables() []string {
	return []string{m.MemoryKey}
}

// LoadMemoryVariables returns the relevant history (summary + recent buffer).
func (m *ConversationSummaryBufferMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	// Corrected: Use GetMessages method
	messages, err := m.ChatHistory.GetMessages(ctx)
	if err != nil {
		return nil, errors.New("error")
	}

	// TODO: Implement token counting and pruning logic based on MaxTokenLimit
	// For now, return summary + all messages (similar to buffer memory)
	bufferString := getBufferString(messages, m.HumanPrefix, m.AiPrefix)
	combinedHistory := m.movingSummaryBuffer + "\n" + bufferString
	if m.movingSummaryBuffer == "" {
		combinedHistory = bufferString
	}

	return map[string]any{m.MemoryKey: combinedHistory}, nil
}

// predictNewSummary uses the LLM to generate a new summary based on existing summary and new lines.
// This is the same helper function as in ConversationSummaryMemory.
func (m *ConversationSummaryBufferMemory) predictNewSummary(ctx context.Context, newLines string) (string, error) {
	promptValue, err := m.SummaryPrompt.Format(ctx, map[string]any{
		"summary":   m.movingSummaryBuffer,
		"new_lines": newLines,
	})
	if err != nil {
		return "", errors.New("error")
	}

	// Pass the prompt value to Invoke
	var promptStr string
	if pv, ok := promptValue.(promptsiface.PromptValue); ok {
		promptStr = pv.ToString()
	} else if str, ok := promptValue.(string); ok {
		promptStr = str
	} else {
		return "", fmt.Errorf("unexpected prompt value type: %T", promptValue)
	}
	llmOutput, err := m.LLM.Invoke(ctx, promptStr)
	if err != nil {
		return "", errors.New("error")
	}

	// Handle different output types
	var newSummary string
	switch output := llmOutput.(type) {
	case string: // Some simple LLMs might return string directly
		newSummary = output
	case schema.Message: // ChatModels will return schema.Message
		newSummary = output.GetContent()
	default:
		return "", errors.New("LLM Invoke returned unexpected type for summary")
	}

	return newSummary, nil
}

// SaveContext adds the latest user input and AI output to the history.
// It potentially triggers summarization if the buffer exceeds the token limit.
func (m *ConversationSummaryBufferMemory) SaveContext(ctx context.Context, inputs, outputs map[string]any) error {
	inputKey := m.InputKey
	outputKey := m.OutputKey

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

	// Corrected: Use imported errors package
	if !inputOk || !outputOk {
		return errors.New("input or output key not found in context maps")
	}

	inputStr, inputStrOk := inputVal.(string)
	outputStr, outputStrOk := outputVal.(string)

	// Corrected: Use imported errors package
	if !inputStrOk || !outputStrOk {
		return errors.New("input or output value is not a string")
	}

	// Add messages to chat history
	err := m.ChatHistory.AddUserMessage(ctx, inputStr)
	if err != nil {
		return errors.New("error")
	}
	err = m.ChatHistory.AddAIMessage(ctx, outputStr)
	if err != nil {
		return errors.New("error")
	}

	// TODO: Implement pruning/summarization logic based on MaxTokenLimit
	// 1. Get all messages from ChatHistory.
	// 2. Calculate total tokens (need a token counting function).
	// 3. If tokens > MaxTokenLimit:
	//    a. Identify messages to summarize (e.g., all but the last K messages).
	//    b. Create 'new_lines' string from messages to summarize.
	//    c. Call predictNewSummary to update m.movingSummaryBuffer.
	//    d. Remove summarized messages from ChatHistory, keeping only the recent buffer.

	return nil
}

// Clear resets the summary and clears the underlying chat history.
func (m *ConversationSummaryBufferMemory) Clear(ctx context.Context) error {
	m.movingSummaryBuffer = ""
	return m.ChatHistory.Clear(ctx)
}

// Ensure ConversationSummaryBufferMemory implements the interface.
var _ iface.Memory = (*ConversationSummaryBufferMemory)(nil)
