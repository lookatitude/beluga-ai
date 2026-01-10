package main

import (
	"context"
	"fmt"
	"log"
	"os"

	_ "github.com/lookatitude/beluga-ai/pkg/llms/providers/openai"
	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools/providers"
	configiface "github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	fmt.Println("🧠 Beluga AI - ReAct Agent Example")
	fmt.Println("===================================")

	ctx := context.Background()

	// Step 1: Create a ChatModel (required for ReAct agents)
	chatLLM, err := createChatModel(ctx)
	if err != nil {
		log.Fatalf("Failed to create chat model: %v", err)
	}

	// Step 2: Create tools for the ReAct agent
	toolList, err := createTools()
	if err != nil {
		log.Fatalf("Failed to create tools: %v", err)
	}

	fmt.Printf("\n📋 Created %d tools for ReAct agent:\n", len(toolList))
	for _, tool := range toolList {
		fmt.Printf("  - %s: %s\n", tool.Name(), tool.Description())
	}

	// Step 3: Create a ReAct agent
	// ReAct agents use a prompt template that guides the reasoning process
	promptTemplate := `You are a helpful assistant that can use tools to answer questions.
When you need to use a tool, think about what tool would be best, then use it.
After using a tool, observe the result and continue reasoning.
When you have enough information, provide a final answer.

Available tools:
{{.tools}}

Question: {{.input}}

Think step by step:`

	reactAgent, err := agents.NewReActAgent(
		"react-assistant",
		chatLLM,
		toolList,
		promptTemplate,
	)
	if err != nil {
		log.Fatalf("Failed to create ReAct agent: %v", err)
	}

	// Step 4: Initialize the agent
	initConfig := map[string]interface{}{
		"max_retries": 3,
		"max_iterations": 10,
	}
	if err := reactAgent.Initialize(initConfig); err != nil {
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	// Step 5: Execute the agent with a complex task
	fmt.Println("\n📝 Executing ReAct agent with complex task...")
	input := map[string]interface{}{
		"input": "What is 25 * 17? Then calculate the square root of that result.",
	}

	result, err := reactAgent.Invoke(ctx, input)
	if err != nil {
		log.Fatalf("Agent execution failed: %v", err)
	}

	// Step 6: Display the result
	fmt.Printf("\n✅ ReAct Agent Response:\n%s\n", result)

	// Step 7: Display agent information
	fmt.Println("\n🤖 Agent Information:")
	fmt.Printf("  Name: %s\n", reactAgent.GetConfig().Name)
	fmt.Printf("  Tools: %d\n", len(reactAgent.GetTools()))
	fmt.Printf("  Health: %v\n", reactAgent.CheckHealth())

	fmt.Println("\n✨ Example completed successfully!")
}

// createChatModel creates a ChatModel instance.
// Uses mock if API key is not set, otherwise uses OpenAI.
func createChatModel(ctx context.Context) (llmsiface.ChatModel, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  OPENAI_API_KEY not set, using mock ChatModel")
		return &mockChatModel{
			modelName:    "mock-chat-model",
			providerName: "mock-provider",
		}, nil
	}

	config := llms.DefaultConfig()
	config.APIKey = apiKey
	config.ModelName = "gpt-3.5-turbo"

	chatModel, err := llms.NewChatModel("gpt-3.5-turbo", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}

	return chatModel, nil
}

// createTools creates a list of tools for the ReAct agent.
func createTools() ([]tools.Tool, error) {
	var toolList []tools.Tool

	// Create calculator tool
	calcConfig := configiface.ToolConfig{
		Name:        "calculator",
		Description: "Performs basic arithmetic operations (add, subtract, multiply, divide)",
	}
	calcTool, err := providers.NewCalculatorTool(calcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create calculator tool: %w", err)
	}
	toolList = append(toolList, calcTool)

	return toolList, nil
}

// mockChatModel is a simple mock implementation for demonstration.
type mockChatModel struct {
	modelName    string
	providerName string
}

func (m *mockChatModel) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	// Mock response simulating ReAct reasoning
	return schema.NewAIMessage("Thought: I need to calculate 25 * 17 first.\nAction: calculator\nAction Input: {\"operation\": \"multiply\", \"a\": 25, \"b\": 17}\nObservation: 425\nThought: Now I need to find the square root of 425.\nAction: calculator\nAction Input: {\"operation\": \"sqrt\", \"value\": 425}\nObservation: 20.615528128088304\nFinal Answer: 25 * 17 = 425, and the square root of 425 is approximately 20.62."), nil
}

func (m *mockChatModel) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error) {
	ch := make(chan llmsiface.AIMessageChunk, 1)
	ch <- llmsiface.AIMessageChunk{
		Content: "Thought: I need to calculate 25 * 17 first.\nAction: calculator\nAction Input: {\"operation\": \"multiply\", \"a\": 25, \"b\": 17}\nObservation: 425\nThought: Now I need to find the square root of 425.\nAction: calculator\nAction Input: {\"operation\": \"sqrt\", \"value\": 425}\nObservation: 20.615528128088304\nFinal Answer: 25 * 17 = 425, and the square root of 425 is approximately 20.62.",
	}
	close(ch)
	return ch, nil
}

func (m *mockChatModel) BindTools(toolsToBind []tools.Tool) llmsiface.ChatModel {
	// Return self - mock doesn't actually bind tools
	return m
}

func (m *mockChatModel) GetModelName() string {
	return m.modelName
}

func (m *mockChatModel) GetProviderName() string {
	return m.providerName
}

func (m *mockChatModel) CheckHealth() map[string]any {
	return map[string]any{"status": "healthy"}
}

func (m *mockChatModel) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := ensureMessages(input)
	if err != nil {
		return nil, err
	}
	response, err := m.Generate(ctx, messages, options...)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (m *mockChatModel) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := m.Invoke(ctx, input, options...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	messages, err := ensureMessages(input)
	if err != nil {
		return nil, err
	}
	chunkChan, err := m.StreamChat(ctx, messages, options...)
	if err != nil {
		return nil, err
	}
	anyChan := make(chan any)
	go func() {
		defer close(anyChan)
		for chunk := range chunkChan {
			anyChan <- chunk
		}
	}()
	return anyChan, nil
}

func ensureMessages(input any) ([]schema.Message, error) {
	switch v := input.(type) {
	case []schema.Message:
		return v, nil
	case map[string]interface{}:
		if inputVal, ok := v["input"].(string); ok {
			return []schema.Message{schema.NewHumanMessage(inputVal)}, nil
		}
		return []schema.Message{schema.NewHumanMessage(fmt.Sprintf("%v", v))}, nil
	case string:
		return []schema.Message{schema.NewHumanMessage(v)}, nil
	default:
		return []schema.Message{schema.NewHumanMessage(fmt.Sprintf("%v", input))}, nil
	}
}
