// Package main demonstrates multimodal agent integration.
// This example shows how to extend agents with multimodal capabilities.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	ctx := context.Background()

	// Example 1: Agent with multimodal image input
	fmt.Println("=== Example 1: Agent with Multimodal Image Input ===")
	agentMultimodalExample(ctx)

	// Example 2: ReAct agent with multimodal capabilities
	fmt.Println("\n=== Example 2: ReAct Agent with Multimodal ===")
	reactAgentMultimodalExample(ctx)
}

func agentMultimodalExample(ctx context.Context) {
	// Create LLM that supports multimodal (using OpenAI)
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Skipping: OPENAI_API_KEY not set")
		return
	}

	config := llms.NewConfig(
		llms.WithProvider("openai"),
		llms.WithModelName("gpt-4o"),
		llms.WithAPIKey(apiKey),
	)

	llm, err := llms.GetRegistry().GetLLM("openai", config)
	if err != nil {
		log.Printf("Failed to create LLM: %v", err)
		return
	}

	// Create agent with multimodal LLM
	agent, err := agents.NewBaseAgent("multimodal-agent", llm, nil)
	if err != nil {
		log.Printf("Failed to create agent: %v", err)
		return
	}

	// Create multimodal message with image
	imageMsg := schema.NewImageMessage("https://example.com/image.png", "What's in this image? Describe what you see.")

	// Process with agent
	response, err := agent.Invoke(ctx, []schema.Message{imageMsg})
	if err != nil {
		log.Printf("Agent invocation failed: %v", err)
		return
	}

	fmt.Printf("Agent Response: %s\n", response)
}

func reactAgentMultimodalExample(ctx context.Context) {
	// Create LLM
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Skipping: OPENAI_API_KEY not set")
		return
	}

	config := llms.NewConfig(
		llms.WithProvider("openai"),
		llms.WithModelName("gpt-4o"),
		llms.WithAPIKey(apiKey),
	)

	chatModel, err := llms.GetRegistry().GetProvider("openai", config)
	if err != nil {
		log.Printf("Failed to create ChatModel: %v", err)
		return
	}

	// Create ReAct agent with multimodal support
	reactAgent, err := agents.NewReActAgent("react-multimodal-agent", chatModel, []tools.Tool{
		// Add tools here if needed
	}, nil)
	if err != nil {
		log.Printf("Failed to create ReAct agent: %v", err)
		return
	}

	// Create multimodal input
	imageMsg := schema.NewImageMessage("https://example.com/image.png", "Analyze this image and describe what you see in detail.")

	// Process with ReAct agent
	response, err := reactAgent.Invoke(ctx, []schema.Message{imageMsg})
	if err != nil {
		log.Printf("ReAct agent invocation failed: %v", err)
		return
	}

	fmt.Printf("ReAct Agent Response: %s\n", response)
}
