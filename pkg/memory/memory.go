package memory

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Memory defines the interface for agent memory systems.
// It allows agents to store and retrieve information from conversations or other sources.
type Memory interface {
	GetMemoryVariables(ctx context.Context, inputs map[string]interface{}) ([]string, error)
	LoadMemoryVariables(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error)
	SaveContext(ctx context.Context, inputs map[string]interface{}, outputs map[string]string) error
	Clear(ctx context.Context) error
	GetMemoryType() string
}

// Config is a generic configuration structure for memory providers.
type Config struct {
	Type         string                 
	Name         string                 
	ProviderArgs map[string]interface{} 
}

// Factory defines the interface for creating Memory instances.
type Factory interface {
	CreateMemory(ctx context.Context, config Config) (Memory, error)
}

// InputOutputKeys defines the keys for input and output variables in memory.
type InputOutputKeys struct {
	InputKey  string
	OutputKey string
}

// DefaultInputKey is the default key for the input variable.
const DefaultInputKey = "input"

// DefaultOutputKey is the default key for the output variable.
const DefaultOutputKey = "output"

// BaseMemory provides a base implementation for memory types.
// It handles common aspects like input/output keys and the memory key.
type BaseMemory struct {
	InputOutputKeys // Embed InputOutputKeys
	MemoryKey       string
}

// NewBaseMemory creates a new BaseMemory instance.
func NewBaseMemory(inputKey, outputKey, memoryKey string) *BaseMemory {
	return &BaseMemory{
		InputOutputKeys: InputOutputKeys{
			InputKey:  inputKey,
			OutputKey: outputKey,
		},
		MemoryKey: memoryKey,
	}
}

// GetMemoryKey returns the memory key for this memory instance.
func (bm *BaseMemory) GetMemoryKey() string {
	return bm.MemoryKey
}

// GetInputOutputKeys determines the input and output keys to use.
// If specific keys are provided, they are used; otherwise, defaults are used.
func GetInputOutputKeys(inputKey, outputKey *string) InputOutputKeys {
	keys := InputOutputKeys{}
	if inputKey != nil {
		keys.InputKey = *inputKey
	} else {
		keys.InputKey = DefaultInputKey
	}
	if outputKey != nil {
		keys.OutputKey = *outputKey
	} else {
		keys.OutputKey = DefaultOutputKey
	}
	return keys
}

// GetPromptInputKey identifies the correct key for the prompt input from a map of inputs.
// It ensures the chosen key is not one of the memory variables.
func GetPromptInputKey(inputs map[string]interface{}, memoryVariables []string, inputKey *string) (string, error) {
	if inputKey != nil {
		// If a specific input key is provided, validate it.
		for _, memVar := range memoryVariables {
			if *inputKey == memVar {
				return "", fmt.Errorf("input key \"%s\" is one of the memory variables %v", *inputKey, memoryVariables)
			}
		}
		if _, ok := inputs[*inputKey]; !ok {
			return "", fmt.Errorf("input key \"%s\" not found in inputs %v", *inputKey, inputs)
		}
		return *inputKey, nil
	}

	// If no input key is provided, try to infer it.
	var candidateKeys []string
	for k := range inputs {
		isMemoryVar := false
		for _, memVar := range memoryVariables {
			if k == memVar {
				isMemoryVar = true
				break
			}
		}
		if !isMemoryVar {
			candidateKeys = append(candidateKeys, k)
		}
	}

	if len(candidateKeys) == 1 {
		return candidateKeys[0], nil
	}
	if len(candidateKeys) == 0 {
		return "", fmt.Errorf("no input keys found that are not memory variables; inputs: %v, memory variables: %v", inputs, memoryVariables)
	}
	return "", fmt.Errorf("multiple input keys found %v; please specify an input key or ensure only one non-memory variable key exists", candidateKeys)
}


// BufferMemory is a simple in-memory buffer for storing conversation history.
type BufferMemory struct {
	ChatHistory    *schema.BaseChatHistory 
	ReturnMessages bool                    
	InputKey       string                  
	OutputKey      string                  
	MemoryKey      string                  
}

// NewBufferMemory creates a new BufferMemory.
func NewBufferMemory(returnMessages bool, inputKey, outputKey, memoryKey string) *BufferMemory {
	if memoryKey == "" {
		memoryKey = "history" 
	}
	return &BufferMemory{
		ChatHistory:    schema.NewBaseChatHistory(), 
		ReturnMessages: returnMessages,
		InputKey:       inputKey,
		OutputKey:      outputKey,
		MemoryKey:      memoryKey,
	}
}

func (bm *BufferMemory) GetMemoryVariables(ctx context.Context, inputs map[string]interface{}) ([]string, error) {
	return []string{bm.MemoryKey}, nil
}

func (bm *BufferMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
	memory := make(map[string]interface{})
	messages, err := bm.ChatHistory.Messages()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve messages from chat history: %w", err)
	}

	if bm.ReturnMessages {
		memory[bm.MemoryKey] = messages
	} else {
		var historyStr string
		for _, msg := range messages {
			historyStr += fmt.Sprintf("%s: %s\n", msg.GetType(), msg.GetContent()) 
		}
		memory[bm.MemoryKey] = historyStr
	}
	return memory, nil
}

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

func (bm *BufferMemory) Clear(ctx context.Context) error {
	bm.ChatHistory = schema.NewBaseChatHistory()
	return nil
}

func (bm *BufferMemory) GetMemoryType() string {
	return "buffer"
}

var _ Memory = (*BufferMemory)(nil)

