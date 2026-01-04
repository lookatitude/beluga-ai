package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools/providers"
	"github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

func main() {
	fmt.Println("🛠️  Beluga AI - Agent with Tools Example")
	fmt.Println("==========================================")

	ctx := context.Background()

	// Step 1: Create an LLM instance
	llm, err := createLLM(ctx)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Step 2: Create tools
	toolList, err := createTools()
	if err != nil {
		log.Fatalf("Failed to create tools: %v", err)
	}

	fmt.Printf("\n📋 Created %d tools:\n", len(toolList))
	for _, tool := range toolList {
		fmt.Printf("  - %s: %s\n", tool.Name(), tool.Description())
	}

	// Step 3: Create an agent with tools
	agent, err := agents.NewBaseAgent("tool-agent", llm, toolList)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Step 4: Initialize the agent
	initConfig := map[string]interface{}{
		"max_retries": 3,
	}
	if err := agent.Initialize(initConfig); err != nil {
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	// Step 5: Execute the agent with a task that requires tools
	fmt.Println("\n📝 Executing agent with task requiring tools...")
	input := map[string]interface{}{
		"input": "Calculate 15 * 23, then echo the result",
	}

	result, err := agent.Invoke(ctx, input)
	if err != nil {
		log.Fatalf("Agent execution failed: %v", err)
	}

	// Step 6: Display the result
	fmt.Printf("\n✅ Agent Response:\n%s\n", result)

	// Step 7: Display available tools
	fmt.Println("\n🔧 Available Tools:")
	for _, tool := range agent.GetTools() {
		fmt.Printf("  - %s: %s\n", tool.Name(), tool.Description())
	}

	fmt.Println("\n✨ Example completed successfully!")
}

// createLLM creates an LLM instance.
// Uses mock if API key is not set, otherwise uses OpenAI.
func createLLM(ctx context.Context) (llmsiface.LLM, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  OPENAI_API_KEY not set, using mock LLM")
		return &mockLLM{
			modelName:    "mock-model",
			providerName: "mock-provider",
		}, nil
	}

	config := llms.NewConfig(
		llms.WithProvider("openai"),
		llms.WithModelName("gpt-3.5-turbo"),
		llms.WithAPIKey(apiKey),
	)

	factory := llms.NewFactory()
	llm, err := factory.CreateProvider("openai", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM: %w", err)
	}

	return llm, nil
}

// createTools creates a list of tools for the agent to use.
func createTools() ([]tools.Tool, error) {
	var toolList []tools.Tool

	// Create calculator tool
	calcConfig := iface.ToolConfig{
		Name:        "calculator",
		Description: "Performs basic arithmetic operations (add, subtract, multiply, divide)",
	}
	calcTool, err := providers.NewCalculatorTool(calcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create calculator tool: %w", err)
	}
	toolList = append(toolList, calcTool)

	// Create echo tool
	echoConfig := iface.ToolConfig{
		Name:        "echo",
		Description: "Echoes back the input text",
	}
	echoTool, err := providers.NewEchoTool(echoConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create echo tool: %w", err)
	}
	toolList = append(toolList, echoTool)

	return toolList, nil
}

// mockLLM is a simple mock implementation for demonstration.
type mockLLM struct {
	modelName    string
	providerName string
}

func (m *mockLLM) Invoke(ctx context.Context, prompt string, callOptions ...interface{}) (string, error) {
	// Mock response that simulates tool usage
	return "I calculated 15 * 23 = 345, and echoed: 345", nil
}

func (m *mockLLM) GetModelName() string {
	return m.modelName
}

func (m *mockLLM) GetProviderName() string {
	return m.providerName
}
