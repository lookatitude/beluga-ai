package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	fmt.Println("🤖 Beluga AI - Basic Agent Example")
	fmt.Println("===================================")

	ctx := context.Background()

	// Step 1: Create an LLM instance
	// For this example, we'll use a mock LLM
	// In production, you would use: llms.NewLLM(ctx, "openai", config)
	llm := createMockLLM()

	// Step 2: Create a basic agent without tools
	agent, err := agents.NewBaseAgent("basic-assistant", llm, nil)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Step 3: Initialize the agent
	initConfig := map[string]interface{}{
		"max_retries": 3,
	}
	if err := agent.Initialize(initConfig); err != nil {
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	// Step 4: Execute the agent with input
	fmt.Println("\n📝 Executing agent with input...")
	input := map[string]interface{}{
		"input": "What is the capital of France?",
	}

	result, err := agent.Invoke(ctx, input)
	if err != nil {
		log.Fatalf("Agent execution failed: %v", err)
	}

	// Step 5: Display the result
	fmt.Printf("\n✅ Agent Response:\n%s\n", result)

	// Step 6: Check agent health
	health := agent.CheckHealth()
	fmt.Printf("\n🏥 Agent Health Status: %v\n", health)

	fmt.Println("\n✨ Example completed successfully!")
}

// createMockLLM creates a simple mock LLM for demonstration purposes.
// In production, replace this with actual LLM initialization:
//
//	config := llms.NewConfig(
//		llms.WithProvider("openai"),
//		llms.WithModelName("gpt-3.5-turbo"),
//		llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
//	)
//	factory := llms.NewFactory()
//	llm, err := factory.CreateProvider("openai", config)
func createMockLLM() iface.LLM {
	return &mockLLM{
		modelName:    "mock-model",
		providerName: "mock-provider",
	}
}

// mockLLM is a simple mock implementation for demonstration.
type mockLLM struct {
	modelName    string
	providerName string
}

func (m *mockLLM) Invoke(ctx context.Context, prompt string, callOptions ...interface{}) (string, error) {
	// Simple mock response
	return "The capital of France is Paris.", nil
}

func (m *mockLLM) GetModelName() string {
	return m.modelName
}

func (m *mockLLM) GetProviderName() string {
	return m.providerName
}

// For production use, uncomment and use this function instead:
func createRealLLM(ctx context.Context) (iface.LLM, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
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
