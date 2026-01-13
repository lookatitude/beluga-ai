package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	fmt.Println("🔄 Beluga AI Memory Package Usage Example")
	fmt.Println("==========================================")

	ctx := context.Background()

	// Example 1: Create Buffer Memory
	fmt.Println("\n📋 Example 1: Creating Buffer Memory")
	bufferMemory, err := memory.NewMemory(memory.MemoryTypeBuffer,
		memory.WithMemoryKey("conversation"),
		memory.WithReturnMessages(true),
	)
	if err != nil {
		log.Fatalf("Failed to create buffer memory: %v", err)
	}
	fmt.Println("✅ Buffer memory created successfully")

	// Example 2: Save Context
	fmt.Println("\n📋 Example 2: Saving Conversation Context")
	inputs := map[string]any{
		"input": "What is machine learning?",
	}
	outputs := map[string]any{
		"output": "Machine learning is a subset of AI that enables systems to learn from data.",
	}
	err = bufferMemory.SaveContext(ctx, inputs, outputs)
	if err != nil {
		log.Fatalf("Failed to save context: %v", err)
	}
	fmt.Println("✅ Context saved successfully")

	// Example 3: Load Memory Variables
	fmt.Println("\n📋 Example 3: Loading Memory Variables")
	newInputs := map[string]any{
		"input": "Can you tell me more?",
	}
	memoryVars, err := bufferMemory.LoadMemoryVariables(ctx, newInputs)
	if err != nil {
		log.Fatalf("Failed to load memory variables: %v", err)
	}
	fmt.Printf("✅ Memory variables loaded: %v\n", memoryVars)

	// Example 4: Create Window Memory
	fmt.Println("\n📋 Example 4: Creating Window Memory")
	history := memory.NewBaseChatMessageHistory()
	_ = memory.NewConversationBufferWindowMemory(history, 3, "history", true)
	fmt.Println("✅ Window memory created (keeps last 3 messages)")

	// Example 5: Add Messages to History
	fmt.Println("\n📋 Example 5: Adding Messages to History")
	history.AddMessage(ctx, schema.NewHumanMessage("Hello"))
	history.AddMessage(ctx, schema.NewAIMessage("Hi! How can I help you?"))
	history.AddMessage(ctx, schema.NewHumanMessage("What is AI?"))
	history.AddMessage(ctx, schema.NewAIMessage("AI is artificial intelligence."))
	fmt.Println("✅ Added 4 messages to history")

	// Example 6: Get Messages
	fmt.Println("\n📋 Example 6: Retrieving Messages")
	messages, err := history.GetMessages(ctx)
	if err != nil {
		log.Fatalf("Failed to get messages: %v", err)
	}
	fmt.Printf("✅ Retrieved %d messages\n", len(messages))
	for i, msg := range messages {
		fmt.Printf("   Message %d: %s - %s\n", i+1, msg.GetType(), msg.GetContent()[:min(50, len(msg.GetContent()))])
	}

	// Example 7: Clear Memory
	fmt.Println("\n📋 Example 7: Clearing Memory")
	err = bufferMemory.Clear(ctx)
	if err != nil {
		log.Fatalf("Failed to clear memory: %v", err)
	}
	fmt.Println("✅ Memory cleared successfully")

	// Example 8: Get Buffer String
	fmt.Println("\n📋 Example 8: Formatting Messages as Buffer String")
	testMessages := []schema.Message{
		schema.NewHumanMessage("What is Python?"),
		schema.NewAIMessage("Python is a programming language."),
	}
	bufferStr := memory.GetBufferString(testMessages, "Human", "AI")
	fmt.Println("✅ Buffer string created:")
	fmt.Printf("   %s", bufferStr)

	fmt.Println("\n✨ All examples completed successfully!")
	fmt.Println("\nFor more examples, see:")
	fmt.Println("  - examples/agents/with_memory/ - Agents with memory")
	fmt.Println("  - examples/rag/with_memory/ - RAG with conversation memory")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
