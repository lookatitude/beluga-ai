package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/memory"
)

func main() {
	fmt.Println("💾 Beluga AI - Agent with Memory Example")
	fmt.Println("=========================================")

	ctx := context.Background()

	// Step 1: Create an LLM instance
	llm, err := createLLM(ctx)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Step 2: Create buffer memory
	// Buffer memory stores all conversation history
	memConfig := memory.Config{
		Type:        memory.MemoryTypeBuffer,
		MemoryKey:   "history",
		InputKey:    "input",
		OutputKey:   "output",
		Enabled:     true,
		ReturnMessages: false,
	}

	memFactory := memory.NewFactory()
	mem, err := memFactory.CreateMemory(ctx, memConfig)
	if err != nil {
		log.Fatalf("Failed to create memory: %v", err)
	}

	fmt.Println("\n💾 Created buffer memory for conversation history")

	// Step 3: Create an agent
	agent, err := agents.NewBaseAgent("memory-agent", llm, nil)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Step 4: Initialize the agent with memory
	initConfig := map[string]interface{}{
		"max_retries": 3,
		"memory":      mem,
	}
	if err := agent.Initialize(initConfig); err != nil {
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	// Step 5: Execute multiple turns of conversation
	conversations := []string{
		"My name is Alice",
		"What is my name?",
		"What did I tell you my name was?",
	}

	fmt.Println("\n💬 Starting multi-turn conversation...")
	for i, userInput := range conversations {
		fmt.Printf("\n--- Turn %d ---\n", i+1)
		fmt.Printf("User: %s\n", userInput)

		// Load memory variables
		inputs := map[string]any{
			"input": userInput,
		}
		memoryVars, err := mem.LoadMemoryVariables(ctx, inputs)
		if err != nil {
			log.Printf("Warning: Failed to load memory: %v", err)
		} else {
			fmt.Printf("Memory context: %v\n", memoryVars)
		}

		// Execute agent
		result, err := agent.Invoke(ctx, inputs)
		if err != nil {
			log.Fatalf("Agent execution failed: %v", err)
		}

		fmt.Printf("Agent: %s\n", result)

		// Save context to memory
		outputs := map[string]any{
			"output": result,
		}
		if err := mem.SaveContext(ctx, inputs, outputs); err != nil {
			log.Printf("Warning: Failed to save context: %v", err)
		}
	}

	// Step 6: Display final memory state
	fmt.Println("\n📚 Final Memory State:")
	memoryVars, err := mem.LoadMemoryVariables(ctx, map[string]any{"input": "summary"})
	if err == nil {
		fmt.Printf("Memory: %v\n", memoryVars)
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

// mockLLM is a simple mock implementation for demonstration.
type mockLLM struct {
	modelName    string
	providerName string
}

func (m *mockLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	// Convert input to string if needed
	var prompt string
	switch v := input.(type) {
	case string:
		prompt = v
	case map[string]interface{}:
		if inputVal, ok := v["input"].(string); ok {
			prompt = inputVal
		} else {
			prompt = fmt.Sprintf("%v", v)
		}
	default:
		prompt = fmt.Sprintf("%v", input)
	}

	// Simple mock responses based on context
	if strings.Contains(prompt, "name is Alice") {
		return "Nice to meet you, Alice! I'll remember that.", nil
	}
	if strings.Contains(prompt, "What is my name") {
		return "Your name is Alice.", nil
	}
	if strings.Contains(prompt, "What did I tell you") {
		return "You told me your name is Alice.", nil
	}
	return "I understand.", nil
}

func (m *mockLLM) GetModelName() string {
	return m.modelName
}

func (m *mockLLM) GetProviderName() string {
	return m.providerName
}
