package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/multimodal"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
)

func main() {
	ctx := context.Background()

	// Example 1: OpenAI multimodal processing
	fmt.Println("=== Example 1: OpenAI Multimodal Processing ===")
	openaiExample(ctx)

	// Example 2: Google Gemini multimodal processing
	fmt.Println("\n=== Example 2: Google Gemini Multimodal Processing ===")
	geminiExample(ctx)

	// Example 3: Text + Image input
	fmt.Println("\n=== Example 3: Text + Image Input ===")
	textImageExample(ctx)

	// Example 4: Streaming multimodal processing
	fmt.Println("\n=== Example 4: Streaming Multimodal Processing ===")
	streamingExample(ctx)
}

func openaiExample(ctx context.Context) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Skipping OpenAI example: OPENAI_API_KEY not set")
		return
	}

	config := multimodal.Config{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   apiKey,
		Timeout:  30 * time.Second,
	}

	model, err := multimodal.NewMultimodalModel(ctx, "openai", config)
	if err != nil {
		log.Printf("Failed to create OpenAI model: %v", err)
		return
	}

	// Create text-only input
	textBlock, err := types.NewContentBlock("text", []byte("What is artificial intelligence?"))
	if err != nil {
		log.Printf("Failed to create text block: %v", err)
		return
	}

	input, err := types.NewMultimodalInput([]*types.ContentBlock{textBlock})
	if err != nil {
		log.Printf("Failed to create input: %v", err)
		return
	}

	output, err := model.Process(ctx, input)
	if err != nil {
		log.Printf("Failed to process: %v", err)
		return
	}

	fmt.Printf("Input ID: %s\n", input.ID)
	fmt.Printf("Output ID: %s\n", output.ID)
	fmt.Printf("Provider: %s\n", output.Provider)
	fmt.Printf("Model: %s\n", output.Model)
	if len(output.ContentBlocks) > 0 {
		fmt.Printf("Response: %s\n", string(output.ContentBlocks[0].Data))
	}
}

func geminiExample(ctx context.Context) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("Skipping Gemini example: GEMINI_API_KEY not set")
		return
	}

	config := multimodal.Config{
		Provider: "gemini",
		Model:    "gemini-1.5-pro",
		APIKey:   apiKey,
		Timeout:  30 * time.Second,
	}

	model, err := multimodal.NewMultimodalModel(ctx, "gemini", config)
	if err != nil {
		log.Printf("Failed to create Gemini model: %v", err)
		return
	}

	// Create text-only input
	textBlock, err := types.NewContentBlock("text", []byte("Explain quantum computing in simple terms."))
	if err != nil {
		log.Printf("Failed to create text block: %v", err)
		return
	}

	input, err := types.NewMultimodalInput([]*types.ContentBlock{textBlock})
	if err != nil {
		log.Printf("Failed to create input: %v", err)
		return
	}

	output, err := model.Process(ctx, input)
	if err != nil {
		log.Printf("Failed to process: %v", err)
		return
	}

	fmt.Printf("Input ID: %s\n", input.ID)
	fmt.Printf("Output ID: %s\n", output.ID)
	fmt.Printf("Provider: %s\n", output.Provider)
	fmt.Printf("Model: %s\n", output.Model)
	if len(output.ContentBlocks) > 0 {
		fmt.Printf("Response: %s\n", string(output.ContentBlocks[0].Data))
	}
}

func textImageExample(ctx context.Context) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Skipping text+image example: OPENAI_API_KEY not set")
		return
	}

	config := multimodal.Config{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   apiKey,
		Timeout:  30 * time.Second,
	}

	model, err := multimodal.NewMultimodalModel(ctx, "openai", config)
	if err != nil {
		log.Printf("Failed to create model: %v", err)
		return
	}

	// Create text block
	textBlock, err := types.NewContentBlock("text", []byte("What's in this image? Describe it in detail."))
	if err != nil {
		log.Printf("Failed to create text block: %v", err)
		return
	}

	// Create image block from URL
	// Note: In a real application, you would use an actual image URL
	imageBlock, err := types.NewContentBlockFromURL(ctx, "image", "https://example.com/sample-image.png")
	if err != nil {
		// If URL fetch fails, create a placeholder
		fmt.Printf("Note: Image URL fetch failed (this is expected in examples): %v\n", err)
		fmt.Println("In a real application, use a valid image URL or provide image data directly.")
		return
	}

	input, err := types.NewMultimodalInput([]*types.ContentBlock{textBlock, imageBlock})
	if err != nil {
		log.Printf("Failed to create input: %v", err)
		return
	}

	output, err := model.Process(ctx, input)
	if err != nil {
		log.Printf("Failed to process: %v", err)
		return
	}

	fmt.Printf("Input ID: %s\n", input.ID)
	fmt.Printf("Output ID: %s\n", output.ID)
	fmt.Printf("Content blocks in input: %d\n", len(input.ContentBlocks))
	fmt.Printf("Content blocks in output: %d\n", len(output.ContentBlocks))
	if len(output.ContentBlocks) > 0 {
		fmt.Printf("Response: %s\n", string(output.ContentBlocks[0].Data))
	}
}

func streamingExample(ctx context.Context) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Skipping streaming example: OPENAI_API_KEY not set")
		return
	}

	config := multimodal.Config{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   apiKey,
		Timeout:  30 * time.Second,
	}

	model, err := multimodal.NewMultimodalModel(ctx, "openai", config)
	if err != nil {
		log.Printf("Failed to create model: %v", err)
		return
	}

	// Create text-only input
	textBlock, err := types.NewContentBlock("text", []byte("Write a short story about a robot learning to paint."))
	if err != nil {
		log.Printf("Failed to create text block: %v", err)
		return
	}

	input, err := types.NewMultimodalInput([]*types.ContentBlock{textBlock})
	if err != nil {
		log.Printf("Failed to create input: %v", err)
		return
	}

	// Process with streaming
	outputChan, err := model.ProcessStream(ctx, input)
	if err != nil {
		log.Printf("Failed to start streaming: %v", err)
		return
	}

	fmt.Println("Streaming response:")
	chunkCount := 0
	for output := range outputChan {
		chunkCount++
		if len(output.ContentBlocks) > 0 {
			// Check if this is an incremental or final chunk
			if incremental, ok := output.ContentBlocks[0].Metadata["incremental"].(bool); ok && incremental {
				// Print incremental updates (you might want to overwrite the same line)
				fmt.Printf("Chunk %d (incremental): %s\n", chunkCount, string(output.ContentBlocks[0].Data))
			} else {
				// Final chunk
				fmt.Printf("Chunk %d (final): %s\n", chunkCount, string(output.ContentBlocks[0].Data))
			}
		}
	}

	fmt.Printf("Received %d chunks\n", chunkCount)
}
