package llms_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/mock"
)

// Example demonstrating basic ChatModel usage
func ExampleNewFactory() {
	// Create a new factory
	factory := llms.NewFactory()

	// Register a mock provider for demonstration
	mockModel := &MockChatModel{modelName: "example-model"}
	factory.RegisterProvider("example", mockModel)

	// Retrieve the provider
	model, err := factory.GetProvider("example")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Retrieved model: %s\n", model.GetModelName())
	// Output: Retrieved model: example-model
}

// Example demonstrating configuration usage
func ExampleNewConfig() {
	// Create configuration with functional options
	config := llms.NewConfig(
		llms.WithProvider("anthropic"),
		llms.WithModelName("claude-3-sonnet-20240229"),
		llms.WithAPIKey("your-api-key"),
		llms.WithTemperatureConfig(0.7),
		llms.WithMaxTokensConfig(1024),
		llms.WithMaxConcurrentBatches(5),
		llms.WithRetryConfig(3, time.Second, 2.0),
	)

	// Validate configuration
	if err := llms.ValidateProviderConfig(context.Background(), config); err != nil {
		log.Printf("Configuration error: %v", err)
		return
	}

	fmt.Printf("Config validated for provider: %s\n", config.Provider)
	// Output: Config validated for provider: anthropic
}

// Example demonstrating error handling
func ExampleLLMError() {
	// Create an LLM error
	err := llms.NewLLMError("generate", llms.ErrCodeRateLimit, fmt.Errorf("rate limit exceeded"))

	// Check if it's an LLM error
	if llms.IsLLMError(err) {
		fmt.Printf("LLM Error Code: %s\n", llms.GetLLMErrorCode(err))
		fmt.Printf("Is Retryable: %t\n", llms.IsRetryableError(err))
	}
	// Output:
	// LLM Error Code: rate_limit
	// Is Retryable: true
}

// Example demonstrating message conversion
func ExampleEnsureMessages() {
	// Convert string to messages
	messages, err := llms.EnsureMessages("Hello, world!")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Converted %d messages\n", len(messages))
	// Output: Converted 1 messages
}

// Example demonstrating streaming (mock implementation)
func ExampleChatModel_StreamChat() {
	// This is a mock example - in real usage, you'd use an actual provider
	mockModel := &MockChatModel{modelName: "streaming-example"}

	// Create mock streaming channel
	streamChan := make(chan iface.AIMessageChunk, 3)
	streamChan <- iface.AIMessageChunk{Content: "Hello"}
	streamChan <- iface.AIMessageChunk{Content: " world"}
	streamChan <- iface.AIMessageChunk{Content: "!"}
	close(streamChan)

	// Note: In a real test, you'd set up mock expectations here
	// For this example, we directly use the mock implementation

	// Use streaming
	messages := []schema.Message{schema.NewHumanMessage("Say hello")}
	resultChan, err := mockModel.StreamChat(context.Background(), messages)
	if err != nil {
		log.Fatal(err)
	}

	// Collect streaming results
	var fullContent string
	for chunk := range resultChan {
		if chunk.Err != nil {
			log.Printf("Stream error: %v", chunk.Err)
			break
		}
		fullContent += chunk.Content
		fmt.Printf("Received: %s\n", chunk.Content)
	}

	fmt.Printf("Full content: %s\n", fullContent)
	// Note: In a real test, you'd assert expectations here
	// Output:
	// Received: Hello
	// Received: world
	// Received: !
	// Full content: Hello world!
}

// Example demonstrating batch processing
func ExampleChatModel_Batch() {
	mockModel := &MockChatModel{modelName: "batch-example"}

	// Mock batch response
	// Note: In a real test, you'd set up mock expectations here

	// Prepare batch inputs
	inputs := []any{
		[]schema.Message{schema.NewHumanMessage("Question 1")},
		[]schema.Message{schema.NewHumanMessage("Question 2")},
		[]schema.Message{schema.NewHumanMessage("Question 3")},
	}

	// Execute batch
	results, err := mockModel.Batch(context.Background(), inputs)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Processed %d items in batch\n", len(results))
	for i, result := range results {
		if msg, ok := result.(schema.Message); ok {
			fmt.Printf("Result %d: %s\n", i+1, msg.GetContent())
		}
	}
	// Note: In a real test, you'd assert expectations here
	// Output:
	// Processed 3 items in batch
	// Result 1: Response 1
	// Result 2: Response 2
	// Result 3: Response 3
}

// Example demonstrating tool binding
func ExampleChatModel_BindTools() {
	mockModel := &MockChatModel{modelName: "tool-example"}

	// Mock tools
	tools := []tools.Tool{
		&MockTool{name: "calculator", description: "Performs calculations"},
		&MockTool{name: "search", description: "Searches the web"},
	}

	// Mock binding behavior
	// Note: In a real test, you'd set up mock expectations here

	// Bind tools
	modelWithTools := mockModel.BindTools(tools)

	fmt.Printf("Model with tools: %s\n", modelWithTools.GetModelName())
	// Note: In a real test, you'd assert expectations here
	// Output: Model with tools: tool-example
}

// Example demonstrating utility functions
func ExampleGetSystemAndHumanPromptsFromSchema() {
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant."),
		schema.NewHumanMessage("What is 2 + 2?"),
		schema.NewAIMessage("4"),
		schema.NewHumanMessage("What is 3 + 3?"),
	}

	system, human := llms.GetSystemAndHumanPromptsFromSchema(messages)

	fmt.Printf("System: %s\n", system)
	fmt.Printf("Human: %s\n", human)
	// Output:
	// System: You are a helpful assistant.
	// Human: What is 2 + 2?
	// What is 3 + 3?
}

// Example demonstrating model validation
func ExampleValidateModelName() {
	// Valid OpenAI model
	err := llms.ValidateModelName("openai", "gpt-4")
	fmt.Printf("OpenAI GPT-4 valid: %t\n", err == nil)

	// Invalid OpenAI model
	err = llms.ValidateModelName("openai", "invalid-model")
	fmt.Printf("OpenAI invalid model: %t\n", err != nil)

	// Valid Anthropic model
	err = llms.ValidateModelName("anthropic", "claude-3-sonnet")
	fmt.Printf("Anthropic Claude valid: %t\n", err == nil)

	// Unknown provider (should pass)
	err = llms.ValidateModelName("unknown", "some-model")
	fmt.Printf("Unknown provider: %t\n", err == nil)
	// Output:
	// OpenAI GPT-4 valid: true
	// OpenAI invalid model: true
	// Anthropic Claude valid: true
	// Unknown provider: true
}

// Mock implementations for examples

type MockChatModel struct {
	mock.Mock
	modelName string
}

func (m *MockChatModel) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	return schema.NewAIMessage("Mock response"), nil
}

func (m *MockChatModel) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	ch := make(chan iface.AIMessageChunk, 1)
	ch <- iface.AIMessageChunk{Content: "Mock stream"}
	close(ch)
	return ch, nil
}

func (m *MockChatModel) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
	return m
}

func (m *MockChatModel) GetModelName() string {
	return m.modelName
}

func (m *MockChatModel) GetProviderName() string {
	return "mock"
}

func (m *MockChatModel) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return "Mock invoke result", nil
}

func (m *MockChatModel) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i := range results {
		results[i] = schema.NewAIMessage(fmt.Sprintf("Batch response %d", i+1))
	}
	return results, nil
}

func (m *MockChatModel) CheckHealth() map[string]interface{} {
	return map[string]interface{}{
		"state":       "healthy",
		"provider":    "mock",
		"model":       m.modelName,
		"timestamp":   int64(1234567890),
		"call_count":  0,
		"tools_count": 0,
		"should_error": false,
		"responses_len": 1,
	}
}

func (m *MockChatModel) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	ch <- "Mock stream result"
	close(ch)
	return ch, nil
}

type MockTool struct {
	name        string
	description string
}

func (m *MockTool) Name() string        { return m.name }
func (m *MockTool) Description() string { return m.description }
func (m *MockTool) Definition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Name:        m.name,
		Description: m.description,
		InputSchema: "{}",
	}
}
func (m *MockTool) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	return "mock tool result", nil
}

func (m *MockTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
	results := make([]any, len(inputs))
	for i := range inputs {
		results[i] = "mock tool batch result"
	}
	return results, nil
}
