package main

import (
	"context"
	"fmt"
	"log"
	"os"

	_ "github.com/lookatitude/beluga-ai/pkg/llms/providers/openai"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	fmt.Println("🤖 Beluga AI - OpenAI LLM Provider Example")
	fmt.Println("===========================================")

	ctx := context.Background()

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create OpenAI provider configuration
	config := &llms.Config{
		APIKey:    apiKey,
		ModelName: "gpt-4",
		BaseURL:   "https://api.openai.com/v1",
	}

	// Create provider using factory
	factory := llms.NewFactory()
	provider, err := factory.CreateProvider("openai", config)
	if err != nil {
		log.Fatalf("Failed to create OpenAI provider: %v", err)
	}
	fmt.Println("✅ Created OpenAI provider")

	// Create messages
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant."),
		schema.NewHumanMessage("What is the capital of France?"),
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
