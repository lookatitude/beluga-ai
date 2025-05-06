// Package memory provides interfaces and implementations for managing conversation history.
package memory

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/llms"
	"github.com/lookatitude/beluga-ai/prompts"
	"github.com/lookatitude/beluga-ai/schema"
)

// DefaultSummaryPrompt is a basic prompt template for summarizing conversations.
// Corrected to use NewStringPromptTemplate
var DefaultSummaryPrompt, _ = prompts.NewStringPromptTemplate(
	"Progressively summarize the lines of conversation provided, adding onto the previous summary returning a new summary.\n\nEXAMPLE\nCurrent summary:\nThe human asks what the AI thinks of artificial intelligence. The AI thinks artificial intelligence is a force for good.\n\nNew lines of conversation:\nHuman: Why do you think artificial intelligence is a force for good?\nAI: Because artificial intelligence will help humans reach their full potential.\n\nNew summary:\nThe human asks what the AI thinks of artificial intelligence. The AI thinks artificial intelligence is a force for good because it will help humans reach their full potential.\nEND OF EXAMPLE\n\nCurrent summary:\n{{.summary}}\n\nNew lines of conversation:\n{{.new_lines}}\n\nNew summary:",
)

// ConversationSummaryMemory summarizes the conversation history over time.
// It keeps a running summary and uses an LLM to condense new interactions into it.
type ConversationSummaryMemory struct {
	ChatHistory    ChatMessageHistory     // Underlying storage for messages (used temporarily before summarization)
	LLM            llms.LLM               // LLM used for generating summaries
	SummaryPrompt  prompts.PromptTemplate // Prompt template used for summarization
	MemoryKey      string                 // Key name for the summary variable in prompts
	InputKey       string                 // Key name for the user input variable (optional, used in SaveContext)
	OutputKey      string                 // Key name for the AI output variable (optional, used in SaveContext)
	HumanPrefix    string                 // Prefix for human messages when creating new lines for summary prompt
	AiPrefix       string                 // Prefix for AI messages when creating new lines for summary prompt
	currentSummary string                 // The current summarized state
	// TODO: Add mutex for concurrent access to currentSummary?
}

// NewConversationSummaryMemory creates a new ConversationSummaryMemory.
func NewConversationSummaryMemory(history ChatMessageHistory, llm llms.LLM, memoryKey string) *ConversationSummaryMemory {
	key := memoryKey
	if key == "" {
		key = "history" // Default memory key, though it holds a summary
	}
	// Handle potential error from NewStringPromptTemplate, though unlikely with static string
	summaryPrompt := DefaultSummaryPrompt
	if summaryPrompt == nil {
		// Fallback or panic, as the default prompt should always parse
		panic("Failed to parse DefaultSummaryPrompt")
	}
	return &ConversationSummaryMemory{
		ChatHistory:   history,
		LLM:           llm,
		SummaryPrompt: summaryPrompt,
		MemoryKey:     key,
		HumanPrefix:   "Human",
		AiPrefix:      "AI",
	}
}

// MemoryVariables returns the key used for the conversation summary.
func (m *ConversationSummaryMemory) MemoryVariables() []string {
	return []string{m.MemoryKey}
}

// LoadMemoryVariables returns the current conversation summary.
func (m *ConversationSummaryMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	// TODO: Add mutex read lock?
	return map[string]any{m.MemoryKey: m.currentSummary}, nil
}

// predictNewSummary uses the LLM to generate a new summary based on existing summary and new lines.
func (m *ConversationSummaryMemory) predictNewSummary(ctx context.Context, newLines string) (string, error) {
	// TODO: Add mutex read lock for m.currentSummary?
	promptValue, err := m.SummaryPrompt.Format(ctx, map[string]any{
		"summary":   m.currentSummary,
		"new_lines": newLines,
	})
	if err != nil {
		return "", fmt.Errorf("failed to format summary prompt: %w", err)
	}

	// Assuming LLM interface takes string prompt
	// Corrected to use ToString()
	llmInput := promptValue.ToString()
	llmOutput, err := m.LLM.Generate(ctx, llmInput)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary with LLM: %w", err)
	}

	// Assuming LLM.Generate returns a string or similar directly
	// Corrected: Check type and extract content
	var newSummary string
	switch output := llmOutput.(type) {
	case string:
		newSummary = output
	case schema.Message:
		newSummary = output.GetContent()
	default:
		return "", fmt.Errorf("LLM returned unexpected type for summary: %T", llmOutput)
	}

	return newSummary, nil
}

// SaveContext adds the latest user input and AI output, then updates the summary.
func (m *ConversationSummaryMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
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

	// Create the string for the new lines
	newLines := GetBufferString([]schema.Message{
		schema.NewHumanMessage(inputStr),
		schema.NewAIMessage(outputStr),
	}, m.HumanPrefix, m.AiPrefix)

	// Predict the new summary
	newSummary, err := m.predictNewSummary(ctx, newLines)
	if err != nil {
		return err // Error already wrapped
	}

	// Update the current summary
	// TODO: Add mutex write lock?
	m.currentSummary = newSummary

	// Optionally, clear the underlying ChatHistory if it was only used temporarily
	// If ChatHistory is meant to persist alongside summary, don t clear.
	// For pure summary memory, clearing might make sense.
	// err = m.ChatHistory.Clear(ctx)
	// if err != nil {
	// 	 return fmt.Errorf("failed to clear temporary chat history: %w", err)
	// }

	return nil
}

// Clear resets the summary and clears the underlying chat history.
func (m *ConversationSummaryMemory) Clear(ctx context.Context) error {
	// TODO: Add mutex write lock?
	m.currentSummary = ""
	return m.ChatHistory.Clear(ctx)
}

// Ensure ConversationSummaryMemory implements the interface.
var _ BaseMemory = (*ConversationSummaryMemory)(nil)
