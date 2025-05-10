package prompts

import (
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// PromptAdapter defines an interface for adapting a generic prompt structure
// or a set of messages into a format suitable for a specific LLM provider or model. // This is crucial because different models (e.g., OpenAI Chat vs. Anthropic Claude)
// have different expectations for how prompts, system messages, and chat history are formatted.
type PromptAdapter interface {
	// Format takes a generic prompt input (which could be a string, a list of schema.Message, or a custom struct)
	// and returns a string or a list of provider-specific message objects suitable for the target LLM.
	// The `inputs` map can contain variables to be interpolated into the prompt template.
	Format(inputs map[string]interface{}) (interface{}, error)

	// GetInputVariables returns a list of variable names that this prompt adapter expects in the input map.
	GetInputVariables() []string
}

// DefaultPromptAdapter is a basic implementation that handles simple string formatting.
// It can be used for models that expect a single string prompt.
type DefaultPromptAdapter struct {
	Template       string   // The prompt template string, e.g., "Translate the following text: {{.text}}"
	InputVariables []string // List of expected input variables, e.g., ["text"]
}

// NewDefaultPromptAdapter creates a new DefaultPromptAdapter.
func NewDefaultPromptAdapter(template string, inputVariables []string) (*DefaultPromptAdapter, error) {
	// TODO: Potentially validate that all inputVariables are present in the template string.
	return &DefaultPromptAdapter{
		Template:       template,
		InputVariables: inputVariables,
	}, nil
}

// Format interpolates the input variables into the template string.
// For this default adapter, it returns a single string.
func (dpa *DefaultPromptAdapter) Format(inputs map[string]interface{}) (interface{}, error) {
	// Basic string replacement for now. A more robust solution would use Go's text/template package.
	// This is a simplified example.
	formattedPrompt := dpa.Template
	for _, key := range dpa.InputVariables {
		value, ok := inputs[key]
		if !ok {
			return "", fmt.Errorf("missing input variable 	%s	 for prompt template", key)
		}
		valStr, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("input variable 	%s	 is not a string (got %T)", key, value)
		}
		// This is a very naive replacement. Using text/template is highly recommended for real use.
		placeholder := fmt.Sprintf("{{.%s}}", key) // Assuming a simple {{.var}} style placeholder
		formattedPrompt = strings.ReplaceAll(formattedPrompt, placeholder, valStr)
	}
	return formattedPrompt, nil
}

// GetInputVariables returns the list of expected input variables.
func (dpa *DefaultPromptAdapter) GetInputVariables() []string {
	return dpa.InputVariables
}

// Ensure DefaultPromptAdapter implements the PromptAdapter interface.
var _ PromptAdapter = (*DefaultPromptAdapter)(nil)

// Helper function to create a schema.Message (useful for chat model adapters)
func NewChatMessage(role schema.MessageType, content string) schema.Message {
	return &schema.ChatMessage{
		BaseMessage: schema.BaseMessage{Content: content},
		Role:        role,
	}
}

// TODO: Implement a ChatPromptAdapter for models that use a list of messages (e.g., OpenAI Chat, Anthropic Claude)
// This adapter would take a system message template, user message template, and potentially AI message templates,
// along with chat history, and format them into the appropriate list of provider-specific message objects.

/*
Example for a future ChatPromptAdapter:

type ChatPromptAdapter struct {
    SystemMessageTemplate *PromptTemplate // Optional
    UserMessageTemplate   *PromptTemplate
    // ... other configurations like how to incorporate chat history
}

func (cpa *ChatPromptAdapter) Format(inputs map[string]interface{}) (interface{}, error) {
    // Logic to format system message, user message, and integrate chat history
    // into a []schema.Message or provider-specific message list.
    var messages []schema.Message

    if cpa.SystemMessageTemplate != nil {
        sysContent, err := cpa.SystemMessageTemplate.Format(inputs)
        if err != nil { return nil, err }
        messages = append(messages, NewChatMessage(schema.ChatMessageTypeSystem, sysContent))
    }

    // Handle chat history from inputs["history"] (e.g., []schema.Message)
    if history, ok := inputs["history"].([]schema.Message); ok {
        messages = append(messages, history...)
    }

    userContent, err := cpa.UserMessageTemplate.Format(inputs)
    if err != nil { return nil, err }
    messages = append(messages, NewChatMessage(schema.ChatMessageTypeUser, userContent))

    return messages, nil
}
*/

