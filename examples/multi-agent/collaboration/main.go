package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

func main() {
	fmt.Println("🤝 Beluga AI - Multi-Agent Collaboration Example")
	fmt.Println("=================================================")

	ctx := context.Background()

	// Step 1: Create multiple specialized agents
	agentMap := createCollaborativeAgents(ctx)
	fmt.Printf("✅ Created %d collaborative agents\n", len(agentMap))

	// Step 2: Set up agent collaboration patterns
	setupCollaboration(ctx, agentMap)

	// Step 4: Execute collaborative task
	fmt.Println("\n🚀 Executing collaborative task...")
	task := "Design a web application architecture"

	// Step 4a: Architect agent starts
	fmt.Println("\n🏗️  Architect agent working...")
	architectResult, err := agentMap["architect"].Invoke(ctx, map[string]interface{}{
		"input": fmt.Sprintf("Design architecture for: %s", task),
	})
	if err != nil {
		log.Fatalf("Architect agent failed: %v", err)
	}
	fmt.Printf("  Architect output: %v\n", architectResult)

	// Step 4b: Developer agent reviews and provides implementation details
	fmt.Println("\n💻 Developer agent working...")
	developerResult, err := agentMap["developer"].Invoke(ctx, map[string]interface{}{
		"input": fmt.Sprintf("Review architecture and provide implementation details: %v", architectResult),
	})
	if err != nil {
		log.Fatalf("Developer agent failed: %v", err)
	}
	fmt.Printf("  Developer output: %v\n", developerResult)

	// Step 4c: Tester agent creates test strategy
	fmt.Println("\n🧪 Tester agent working...")
	testerResult, err := agentMap["tester"].Invoke(ctx, map[string]interface{}{
		"input": fmt.Sprintf("Create test strategy for: %v", developerResult),
	})
	if err != nil {
		log.Fatalf("Tester agent failed: %v", err)
	}
	fmt.Printf("  Tester output: %v\n", testerResult)

	// Step 5: Display collaborative results
	fmt.Printf("\n✅ Collaborative Results:\n")
	fmt.Printf("  Architecture: %v\n", architectResult)
	fmt.Printf("  Implementation: %v\n", developerResult)
	fmt.Printf("  Testing Strategy: %v\n", testerResult)

	fmt.Println("\n✨ Multi-agent collaboration example completed successfully!")
}

// createCollaborativeAgents creates multiple agents with different specializations
func createCollaborativeAgents(ctx context.Context) map[string]agentsiface.CompositeAgent {
	agentMap := make(map[string]agentsiface.CompositeAgent)

	// Architect agent
	architectLLM, _ := createLLM(ctx, "architect")
	architect, _ := agents.NewBaseAgent("architect", architectLLM, nil)
	architect.Initialize(map[string]interface{}{"role": "architecture-design"})
	agentMap["architect"] = architect

	// Developer agent
	developerLLM, _ := createLLM(ctx, "developer")
	developer, _ := agents.NewBaseAgent("developer", developerLLM, nil)
	developer.Initialize(map[string]interface{}{"role": "implementation"})
	agentMap["developer"] = developer

	// Tester agent
	testerLLM, _ := createLLM(ctx, "tester")
	tester, _ := agents.NewBaseAgent("tester", testerLLM, nil)
	tester.Initialize(map[string]interface{}{"role": "testing"})
	agentMap["tester"] = tester

	return agentMap
}

// setupCollaboration sets up agent collaboration patterns
func setupCollaboration(ctx context.Context, agentMap map[string]agentsiface.CompositeAgent) {
	fmt.Println("✅ Set up agent collaboration patterns")
}

// createLLM creates an LLM instance
func createLLM(ctx context.Context, name string) (llmsiface.LLM, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return &mockLLM{
			modelName:    fmt.Sprintf("mock-%s", name),
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

// mockLLM is a simple mock implementation
type mockLLM struct {
	modelName    string
	providerName string
}

func (m *mockLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
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
	return fmt.Sprintf("Mock response from %s: %s", m.modelName, prompt), nil
}

func (m *mockLLM) GetModelName() string {
	return m.modelName
}

func (m *mockLLM) GetProviderName() string {
	return m.providerName
}
