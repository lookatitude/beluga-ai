package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	_ "github.com/lookatitude/beluga-ai/pkg/llms/providers/anthropic"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	fmt.Println("🤖 Beluga AI - Anthropic Claude LLM Provider Example")
	fmt.Println("======================================================")

	ctx := context.Background()

	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	// Create Anthropic provider configuration
	config := &llms.Config{
		APIKey:    apiKey,
		ModelName: "claude-3-opus-20240229",
		BaseURL:   "https://api.anthropic.com",
	}

	// Create provider using factory
	factory := llms.NewFactory()
	provider, err := factory.CreateProvider("anthropic", config)
	if err != nil {
		log.Fatalf("Failed to create Anthropic provider: %v", err)
	}
	fmt.Println("✅ Created Anthropic provider")

	// Create messages
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant."),
		schema.NewHumanMessage("Explain quantum computing in simple terms."),
	}

	// Generate response
	fmt.Println("\n📝 Generating response...")
	response, err := provider.Generate(ctx, messages)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}

	fmt.Printf("✅ Response: %s\n", response.GetContent())
	fmt.Println("\n✨ Example completed successfully!")
}
