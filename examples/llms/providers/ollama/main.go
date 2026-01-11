package main

import (
	"context"
	"fmt"
	"log"

	_ "github.com/lookatitude/beluga-ai/pkg/llms/providers/ollama"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	fmt.Println("🤖 Beluga AI - Ollama LLM Provider Example")
	fmt.Println("===========================================")

	ctx := context.Background()

	// Create Ollama provider configuration
	// Note: Ollama runs locally, no API key needed
	config := &llms.Config{
		ModelName: "llama2",
		// Optional: Set custom base URL if Ollama is not on localhost:11434
		ProviderSpecific: map[string]any{
			"base_url": "http://localhost:11434",
		},
	}

	// Create provider using factory
	factory := llms.NewFactory()
	provider, err := factory.CreateProvider("ollama", config)
	if err != nil {
		log.Fatalf("Failed to create Ollama provider: %v", err)
	}
	fmt.Println("✅ Created Ollama provider")

	// Create messages
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant."),
		schema.NewHumanMessage("Write a haiku about programming."),
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
