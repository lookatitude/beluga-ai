package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/orchestration"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/internal/messagebus"
)

func main() {
	fmt.Println("👥 Beluga AI - Multi-Agent Orchestration Example")
	fmt.Println("==================================================")

	ctx := context.Background()

	// Step 1: Create LLMs for different agents
	llm1, err := createLLM(ctx, "agent-1-llm")
	if err != nil {
		log.Fatalf("Failed to create LLM 1: %v", err)
	}

	llm2, err := createLLM(ctx, "agent-2-llm")
	if err != nil {
		log.Fatalf("Failed to create LLM 2: %v", err)
	}

	llm3, err := createLLM(ctx, "agent-3-llm")
	if err != nil {
		log.Fatalf("Failed to create LLM 3: %v", err)
	}

	// Step 2: Create multiple agents with different roles
	agent1, err := agents.NewBaseAgent("researcher", llm1, nil)
	if err != nil {
		log.Fatalf("Failed to create agent 1: %v", err)
	}

	agent2, err := agents.NewBaseAgent("analyzer", llm2, nil)
	if err != nil {
		log.Fatalf("Failed to create agent 2: %v", err)
	}

	agent3, err := agents.NewBaseAgent("synthesizer", llm3, nil)
	if err != nil {
		log.Fatalf("Failed to create agent 3: %v", err)
	}

	fmt.Println("✅ Created 3 agents with different roles")

	// Step 3: Create a message bus for agent communication
	messageBus := messagebus.NewInMemoryMessageBus()
	fmt.Println("✅ Created message bus for agent communication")

	// Step 4: Set up agent communication
	// Agent 1 publishes research results
	// Agent 2 subscribes to research results and publishes analysis
	// Agent 3 subscribes to analysis and produces final synthesis

	// Subscribe agents to relevant topics
	researchTopic := "research.results"
	analysisTopic := "analysis.results"

	// Agent 2 subscribes to research results
	_, err = messageBus.Subscribe(ctx, researchTopic, func(ctx context.Context, message interface{}) error {
		fmt.Printf("  [Analyzer] Received research result: %v\n", message)
		// Agent 2 would process and publish analysis
		return messageBus.Publish(ctx, analysisTopic, map[string]interface{}{
			"analysis": "Analysis of research results",
			"source":   message,
		})
	})
	if err != nil {
		log.Fatalf("Failed to subscribe agent 2: %v", err)
	}

	// Agent 3 subscribes to analysis results
	_, err = messageBus.Subscribe(ctx, analysisTopic, func(ctx context.Context, message interface{}) error {
		fmt.Printf("  [Synthesizer] Received analysis: %v\n", message)
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to subscribe agent 3: %v", err)
	}

	// Step 5: Create a chain that coordinates the agents
	agentSteps := []interface{}{
		&agentRunnable{agent: agent1, name: "researcher"},
		&agentRunnable{agent: agent2, name: "analyzer"},
		&agentRunnable{agent: agent3, name: "synthesizer"},
	}

	fmt.Println("\n🔗 Created agent coordination chain")

	// Step 6: Execute multi-agent workflow
	fmt.Println("\n🚀 Executing multi-agent workflow...")

	// Step 6a: Agent 1 performs research
	fmt.Println("\n📚 Step 1: Researcher agent working...")
	input1 := map[string]interface{}{
		"input": "Research topic: Artificial Intelligence trends",
	}
	result1, err := agent1.Invoke(ctx, input1)
	if err != nil {
		log.Fatalf("Agent 1 execution failed: %v", err)
	}
	fmt.Printf("  Research result: %v\n", result1)

	// Publish research result
	err = messageBus.Publish(ctx, researchTopic, result1)
	if err != nil {
		log.Fatalf("Failed to publish research result: %v", err)
	}

	// Step 6b: Agent 2 analyzes (triggered by message bus)
	fmt.Println("\n🔍 Step 2: Analyzer agent working...")
	input2 := map[string]interface{}{
		"input": fmt.Sprintf("Analyze: %v", result1),
	}
	result2, err := agent2.Invoke(ctx, input2)
	if err != nil {
		log.Fatalf("Agent 2 execution failed: %v", err)
	}
	fmt.Printf("  Analysis result: %v\n", result2)

	// Step 6c: Agent 3 synthesizes
	fmt.Println("\n📝 Step 3: Synthesizer agent working...")
	input3 := map[string]interface{}{
		"input": fmt.Sprintf("Synthesize research and analysis: %v + %v", result1, result2),
	}
	result3, err := agent3.Invoke(ctx, input3)
	if err != nil {
		log.Fatalf("Agent 3 execution failed: %v", err)
	}
	fmt.Printf("  Final synthesis: %v\n", result3)

	// Step 7: Display final result
	fmt.Printf("\n✅ Multi-Agent Workflow Result:\n")
	fmt.Printf("  Research: %v\n", result1)
	fmt.Printf("  Analysis: %v\n", result2)
	fmt.Printf("  Synthesis: %v\n", result3)

	fmt.Println("\n✨ Multi-agent orchestration example completed successfully!")
}

// agentRunnable wraps an agent as a Runnable for chain execution
type agentRunnable struct {
	agent interface{}
	name  string
}

// createLLM creates an LLM instance
func createLLM(ctx context.Context, name string) (llmsiface.LLM, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Printf("⚠️  OPENAI_API_KEY not set, using mock LLM for %s\n", name)
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

func (m *mockLLM) Invoke(ctx context.Context, prompt string, callOptions ...interface{}) (string, error) {
	return fmt.Sprintf("Mock response from %s", m.modelName), nil
}

func (m *mockLLM) GetModelName() string {
	return m.modelName
}

func (m *mockLLM) GetProviderName() string {
	return m.providerName
}
