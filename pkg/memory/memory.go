package memory

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Memory defines the interface for agent memory systems.
// It allows agents to store and retrieve information from conversations or other sources.
type Memory interface {
	// GetMemoryVariables returns the keys of the variables that this memory class will return.
	// For example, a BufferMemory might return {"history": "..."}.
	GetMemoryVariables(ctx context.Context, inputs map[string]interface{}) ([]string, error)

	// LoadMemoryVariables retrieves the context from memory.
	// The `inputs` map can provide additional context if needed by the memory type (e.g., session ID).
	LoadMemoryVariables(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error)

	// SaveContext stores the given context into memory.
	// `inputs` are the original inputs to the LLM call, `outputs` are the LLM's response.
	SaveContext(ctx context.Context, inputs map[string]interface{}, outputs map[string]string) error

	// Clear removes all data from the memory.
	Clear(ctx context.Context) error

	// GetMemoryType returns a string identifier for the type of memory (e.g., "buffer", "vector_store").
	GetMemoryType() string
}

// Config is a generic configuration structure for memory providers.
// Specific providers can embed this and add their own fields.
type Config struct {
	Type         string                 // e.g., "buffer", "vector_store", "dynamodb_summary"
	Name         string                 // A unique name for this memory configuration instance (optional, for lookup)
	ProviderArgs map[string]interface{} // Provider-specific arguments
}

// Factory defines the interface for creating Memory instances.
type Factory interface {
	CreateMemory(ctx context.Context, config Config) (Memory, error)
}

// BufferMemory is a simple in-memory buffer for storing conversation history.
type BufferMemory struct {
	ChatHistory    *schema.BaseChatHistory // Uses the BaseChatHistory struct from the schema package
	ReturnMessages bool                    // If true, LoadMemoryVariables returns schema.Message objects, otherwise a formatted string.
	InputKey       string                  // Key for the input variable in SaveContext, e.g., "input"
	OutputKey      string                  // Key for the output variable in SaveContext, e.g., "output"
	MemoryKey      string                  // Key under which the memory is stored and retrieved, e.g., "history"
}

// NewBufferMemory creates a new BufferMemory.
func NewBufferMemory(returnMessages bool, inputKey, outputKey, memoryKey string) *BufferMemory {
	if memoryKey == "" {
		memoryKey = "history" // Default memory key
	}
	return &BufferMemory{
		ChatHistory:    schema.NewBaseChatHistory(), // Initialize with empty history
		ReturnMessages: returnMessages,
		InputKey:       inputKey,
		OutputKey:      outputKey,
		MemoryKey:      memoryKey,
	}
}

// GetMemoryVariables returns the key for the chat history.
func (bm *BufferMemory) GetMemoryVariables(ctx context.Context, inputs map[string]interface{}) ([]string, error) {
	return []string{bm.MemoryKey}, nil
}

// LoadMemoryVariables retrieves the chat history.
func (bm *BufferMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
	memory := make(map[string]interface{})
	messages, err := bm.ChatHistory.Messages()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve messages from chat history: %w", err)
	}

	if bm.ReturnMessages {
		memory[bm.MemoryKey] = messages
	} else {
		// Format messages into a single string (simple concatenation for now)
		var historyStr string
		for _, msg := range messages {
			// Ensure msg.Type and msg.Content are valid; msg.String() might be safer if it handles nil/empty gracefully
			historyStr += fmt.Sprintf("%s: %s\n", msg.GetType(), msg.GetContent()) // Use GetType() and GetContent()
		}
		memory[bm.MemoryKey] = historyStr
	}
	return memory, nil
}

// SaveContext adds the input and output to the chat history.
func (bm *BufferMemory) SaveContext(ctx context.Context, inputs map[string]interface{}, outputs map[string]string) error {
	inputVal, okInput := inputs[bm.InputKey].(string)
	outputVal, okOutput := outputs[bm.OutputKey]

	if !okInput {
		return fmt.Errorf("input key '%s' not found in inputs or not a string", bm.InputKey)
	}
	if !okOutput {
		return fmt.Errorf("output key '%s' not found in outputs", bm.OutputKey)
	}

	err := bm.ChatHistory.AddUserMessage(inputVal)
	if err != nil {
		return fmt.Errorf("failed to add user message to chat history: %w", err)
	}
	err = bm.ChatHistory.AddAIMessage(outputVal)
	if err != nil {
		return fmt.Errorf("failed to add AI message to chat history: %w", err)
	}
	return nil
}

// Clear resets the chat history.
func (bm *BufferMemory) Clear(ctx context.Context) error {
	bm.ChatHistory = schema.NewBaseChatHistory()
	return nil
}

// GetMemoryType returns the type of this memory.
func (bm *BufferMemory) GetMemoryType() string {
	return "buffer"
}

// Ensure BufferMemory implements the Memory interface.
var _ Memory = (*BufferMemory)(nil)

