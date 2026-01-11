package main

import (
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	fmt.Println("🔄 Beluga AI Schema Package Usage Example")
	fmt.Println("==========================================")

	// Example 1: Create Human Message
	fmt.Println("\n📋 Example 1: Creating Human Message")
	humanMsg := schema.NewHumanMessage("What is artificial intelligence?")
	fmt.Printf("✅ Created human message: %s\n", humanMsg.GetContent())
	fmt.Printf("   Type: %s\n", humanMsg.GetType())

	// Example 2: Create AI Message
	fmt.Println("\n📋 Example 2: Creating AI Message")
	aiMsg := schema.NewAIMessage("Artificial intelligence is the simulation of human intelligence by machines.")
	fmt.Printf("✅ Created AI message: %s\n", aiMsg.GetContent())
	fmt.Printf("   Type: %s\n", aiMsg.GetType())

	// Example 3: Create System Message
	fmt.Println("\n📋 Example 3: Creating System Message")
	systemMsg := schema.NewSystemMessage("You are a helpful assistant.")
	fmt.Printf("✅ Created system message: %s\n", systemMsg.GetContent())
	fmt.Printf("   Type: %s\n", systemMsg.GetType())

	// Example 4: Create Document
	fmt.Println("\n📋 Example 4: Creating Document")
	doc := schema.NewDocument(
		"Machine learning is a subset of artificial intelligence.",
		map[string]string{
			"source": "textbook",
			"topic":  "AI",
		},
	)
	fmt.Printf("✅ Created document: %s\n", doc.GetContent())
	fmt.Printf("   Metadata: %v\n", doc.GetMetadata())

	// Example 5: Create Image Message
	fmt.Println("\n📋 Example 5: Creating Image Message")
	imageMsg, err := schema.NewImageMessage("https://example.com/image.jpg", "image/jpeg")
	if err != nil {
		log.Printf("⚠️  Failed to create image message: %v", err)
	} else {
		fmt.Printf("✅ Created image message\n")
		fmt.Printf("   URL: %s\n", imageMsg.GetURL())
		fmt.Printf("   MIME Type: %s\n", imageMsg.GetMIMEType())
	}

	// Example 6: Create Voice Document
	fmt.Println("\n📋 Example 6: Creating Voice Document")
	voiceDoc, err := schema.NewVoiceDocument(
		"https://example.com/audio.wav",
		"audio/wav",
		map[string]string{"duration": "10s"},
	)
	if err != nil {
		log.Printf("⚠️  Failed to create voice document: %v", err)
	} else {
		fmt.Printf("✅ Created voice document\n")
		fmt.Printf("   URL: %s\n", voiceDoc.GetURL())
		fmt.Printf("   MIME Type: %s\n", voiceDoc.GetMIMEType())
	}

	// Example 7: Message Collection
	fmt.Println("\n📋 Example 7: Working with Message Collections")
	messages := []schema.Message{
		humanMsg,
		aiMsg,
		systemMsg,
	}
	fmt.Printf("✅ Created message collection with %d messages\n", len(messages))
	for i, msg := range messages {
		fmt.Printf("   Message %d: %s - %s\n", i+1, msg.GetType(), msg.GetContent()[:min(50, len(msg.GetContent()))])
	}

	fmt.Println("\n✨ All examples completed successfully!")
	fmt.Println("\nFor more examples, see:")
	fmt.Println("  - examples/llm-usage/ - LLM usage with messages")
	fmt.Println("  - examples/rag/simple/ - RAG with documents")
	fmt.Println("  - Package documentation: pkg/schema/README.md")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
