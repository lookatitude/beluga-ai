// Package memory provides interfaces and implementations for managing conversation history.
package memory

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/core" // Import core for Runnable
	"github.com/lookatitude/beluga-ai/prompts"
	"github.com/lookatitude/beluga-ai/schema"
)

// DefaultSummaryPrompt is a basic prompt template for summarizing conversations.
var DefaultSummaryPrompt, _ = prompts.NewStringPromptTemplate(
	"Progressively summarize the lines of conversation provided, adding onto the previous summary returning a new summary.\n\nEXAMPLE\nCurrent summary:\nThe human asks what the AI thinks of artificial intelligence. The AI thinks artificial intelligence is a force for good.\n\nNew lines of conversation:\nHuman: Why do you think artificial intelligence is a force for good?\nAI: Because artificial intelligence will help humans reach their full potential.\n\nNew summary:\nThe human asks what the AI thinks of artificial intelligence. The AI thinks artificial intelligence is a force for good because it will help humans reach their full potential.\nEND OF EXAMPLE\n\nCurrent summary:\n{{.summary}}\n\nNew lines of conversation:\n{{.new_lines}}\n\nNew summary:",
)

// ConversationSummaryMemory summarizes the conversation history over time.
// It keeps a running summary and uses an LLM to condense new interactions into it.
type ConversationSummaryMemory struct {
	ChatHistory    ChatMessageHistory     // Underlying storage for messages (used temporarily before summarization)
	LLM            core.Runnable          // LLM used for generating summaries, changed from llms.LLM to core.Runnable
	SummaryPrompt  prompts.PromptTemplate // Prompt template used for summarization
	MemoryKey      string                 // Key name for the summary variable in prompts
	InputKey       string                 // Key name for the user input variable (optional, used in SaveContext)
	OutputKey      string                 // Key name for the AI output variable (optional, used in SaveContext)
	HumanPrefix    string                 // Prefix for human messages when creating new lines for summary prompt
	AiPrefix       string                 // Prefix for AI messages when creating new lines for summary prompt
	currentSummary string                 // The current summarized state
}

// NewConversationSummaryMemory creates a new ConversationSummaryMemory.
func NewConversationSummaryMemory(history ChatMessageHistory, llm core.Runnable, memoryKey string) *ConversationSummaryMemory {
	key := memoryKey
	if key == "" {
		key = "history" 
	}
	summaryPrompt := DefaultSummaryPrompt
	if summaryPrompt == nil {
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
	return map[string]any{m.MemoryKey: m.currentSummary}, nil
}

// predictNewSummary uses the LLM to generate a new summary based on existing summary and new lines.
func (m *ConversationSummaryMemory) predictNewSummary(ctx context.Context, newLines string) (string, error) {
	promptValue, err := m.SummaryPrompt.Format(ctx, map[string]any{
		"summary":   m.currentSummary,
		"new_lines": newLines,
	})
	if err != nil {
		return "", fmt.Errorf("failed to format summary prompt: %w", err)
	}

	// LLM is now core.Runnable, so we use Invoke.
	// The input to Invoke should be compatible with what the underlying LLM (e.g., ChatModel) expects.
	// For summarization, this is typically a list of messages or a direct prompt string.
	// We are passing a string prompt (promptValue.ToString()) to a ChatModel's Invoke method.
	// This will be handled by llms.EnsureMessages in the ChatModel's Invoke.
	llmOutput, err := m.LLM.Invoke(ctx, promptValue.ToString())
	if err != nil {
		return "", fmt.Errorf("failed to generate summary with LLM: %w", err)
	}

	var newSummary string
	switch output := llmOutput.(type) {
	case string: // Some simple LLMs might return string directly
		newSummary = output
	case schema.Message: // ChatModels will return schema.Message
		newSummary = output.GetContent()
	default:
		return "", fmt.Errorf("LLM Invoke returned unexpected type for summary: %T", llmOutput)
	}

	return newSummary, nil
}

// SaveContext adds the latest user input and AI output, then updates the summary.
func (m *ConversationSummaryMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	inputKey := m.InputKey
	outputKey := m.OutputKey
	var err error

	if inputKey == "" || outputKey == "" {
		inputKey, outputKey, err = GetInputOutputKeys(inputs, outputs)
		if err != nil {
			return fmt.Errorf("failed to determine input/output keys for saving context: %w", err)
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

	newLines := GetBufferString([]schema.Message{
		schema.NewHumanMessage(inputStr),
		schema.NewAIMessage(outputStr),
	}, m.HumanPrefix, m.AiPrefix)

	newSummary, err := m.predictNewSummary(ctx, newLines)
	if err != nil {
		return err
	}

	m.currentSummary = newSummary
	return nil
}

// Clear resets the summary and clears the underlying chat history.
func (m *ConversationSummaryMemory) Clear(ctx context.Context) error {
	m.currentSummary = ""
	return m.ChatHistory.Clear(ctx)
}

// Ensure ConversationSummaryMemory implements the interface.
var _ BaseMemory = (*ConversationSummaryMemory)(nil)

