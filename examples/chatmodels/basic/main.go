package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/pkg/chatmodels"
	_ "github.com/lookatitude/beluga-ai/pkg/chatmodels/providers/openai"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	fmt.Println("💬 Beluga AI - ChatModels Package Example")
	fmt.Println("==========================================")

	ctx := context.Background()

	// Step 1: Create chat model configuration
	fmt.Println("\n📋 Step 1: Creating chat model configuration...")
	config := chatmodels.NewDefaultConfig()
	config.DefaultProvider = "openai"
	config.DefaultModel = "gpt-4"
	fmt.Println("✅ Configuration created")

	// Step 2: Create chat model
	fmt.Println("\n📋 Step 2: Creating chat model...")
	model, err := chatmodels.NewChatModel("gpt-4", config)
	if err != nil {
		log.Printf("Note: Chat model creation may require API keys: %v", err)
		log.Println("Using mock model for demonstration...")
		model, err = chatmodels.NewMockChatModel("gpt-4")
		if err != nil {
			log.Fatalf("Failed to create mock model: %v", err)
		}
	}
	fmt.Println("✅ Chat model created")

	// Step 3: Create messages
	fmt.Println("\n📋 Step 3: Creating messages...")
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant."),
		schema.NewHumanMessage("What is the capital of France?"),
	}
	fmt.Println("✅ Messages created")

	// Step 4: Generate response
	fmt.Println("\n📋 Step 4: Generating response...")
	response, err := model.GenerateMessages(ctx, messages)
	if err != nil {
		log.Fatalf("Failed to generate messages: %v", err)
	}
	fmt.Printf("✅ Response: %s\n", response[0].GetContent())

	// Step 5: Stream messages (optional)
	fmt.Println("\n📋 Step 5: Streaming messages...")
	stream, err := model.StreamMessages(ctx, messages)
	if err != nil {
		log.Printf("Note: Streaming may not be available: %v", err)
	} else {
		fmt.Println("✅ Streaming started")
		for msg := range stream {
			fmt.Printf("  Received: %s\n", msg.GetContent())
		}
	}

	fmt.Println("\n✨ Example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Configure API keys for real providers")
	fmt.Println("- Use custom temperature and max tokens")
	fmt.Println("- Enable function calling for tool use")
}
