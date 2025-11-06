// Package summary provides summary-based memory implementations.
// It contains implementations that condense conversations using LLMs.
package summary

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/prompts"
	promptsiface "github.com/lookatitude/beluga-ai/pkg/prompts/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// DefaultSummaryPrompt is a basic prompt template for summarizing conversations.
var DefaultSummaryPrompt, _ = prompts.NewStringPromptTemplate(
	"default_summary",
	"Progressively summarize the lines of conversation provided, adding onto the previous summary returning a new summary.\n\nEXAMPLE\nCurrent summary:\nThe human asks what the AI thinks of artificial intelligence. The AI thinks artificial intelligence is a force for good.\n\nNew lines of conversation:\nHuman: Why do you think artificial intelligence is a force for good?\nAI: Because artificial intelligence will help humans reach their full potential.\n\nNew summary:\nThe human asks what the AI thinks of artificial intelligence. The AI thinks artificial intelligence is a force for good because it will help humans reach their full potential.\nEND OF EXAMPLE\n\nCurrent summary:\n{{.summary}}\n\nNew lines of conversation:\n{{.new_lines}}\n\nNew summary:",
)

// ConversationSummaryMemory summarizes the conversation history over time.
// It keeps a running summary and uses an LLM to condense new interactions into it.
type ConversationSummaryMemory struct {
	ChatHistory    iface.ChatMessageHistory // Underlying storage for messages (used temporarily before summarization)
	LLM            core.Runnable            // LLM used for generating summaries
	SummaryPrompt  promptsiface.Template    // Prompt template used for summarization
	MemoryKey      string                   // Key name for the summary variable in prompts
	InputKey       string                   // Key name for the user input variable (optional, used in SaveContext)
	OutputKey      string                   // Key name for the AI output variable (optional, used in SaveContext)
	HumanPrefix    string                   // Prefix for human messages when creating new lines for summary prompt
	AiPrefix       string                   // Prefix for AI messages when creating new lines for summary prompt
	currentSummary string                   // The current summarized state
}

// NewConversationSummaryMemory creates a new ConversationSummaryMemory.
func NewConversationSummaryMemory(history iface.ChatMessageHistory, llm core.Runnable, memoryKey string) *ConversationSummaryMemory {
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
		return "", fmt.Errorf("error")
	}

	// LLM is now core.Runnable, so we use Invoke.
	// The input to Invoke should be compatible with what the underlying LLM (e.g., ChatModel) expects.
	// For summarization, this is typically a list of messages or a direct prompt string.
	// We are passing a string prompt (promptValue.ToString()) to a ChatModel's Invoke method.
	// This will be handled by llms.EnsureMessages in the ChatModel's Invoke.
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
		return "", fmt.Errorf("error")
	}

	var newSummary string
	switch output := llmOutput.(type) {
	case string: // Some simple LLMs might return string directly
		newSummary = output
	case schema.Message: // ChatModels will return schema.Message
		newSummary = output.GetContent()
	default:
		return "", fmt.Errorf(
			"LLM Invoke returned unexpected type for summary")
	}

	return newSummary, nil
}

// SaveContext adds the latest user input and AI output, then updates the summary.
func (m *ConversationSummaryMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
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

	if !inputOk {
		return fmt.Errorf(
			"error")
	}
	if !outputOk {
		return fmt.Errorf(
			"error")
	}

	inputStr, inputStrOk := inputVal.(string)
	outputStr, outputStrOk := outputVal.(string)

	if !inputStrOk {
		return fmt.Errorf(
			"error")
	}
	if !outputStrOk {
		return fmt.Errorf(
			"error")
	}

	newLines := getBufferString([]schema.Message{
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

// Ensure ConversationSummaryMemory implements the interface.
var _ iface.Memory = (*ConversationSummaryMemory)(nil)
