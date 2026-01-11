// Package main demonstrates multimodal reasoning and generation capabilities.
// This example shows visual question answering, image captioning, and multimodal reasoning.
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

	// Example 1: Visual Question Answering
	fmt.Println("=== Example 1: Visual Question Answering ===")
	visualQAExample(ctx)

	// Example 2: Image Captioning
	fmt.Println("\n=== Example 2: Image Captioning ===")
	imageCaptioningExample(ctx)

	// Example 3: Multimodal Reasoning
	fmt.Println("\n=== Example 3: Multimodal Reasoning ===")
	multimodalReasoningExample(ctx)
}

func visualQAExample(ctx context.Context) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Skipping: OPENAI_API_KEY not set")
		return
	}

	model, err := multimodal.NewMultimodalModel(ctx, "openai", multimodal.Config{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   apiKey,
		Timeout:  30 * time.Second,
	})
	if err != nil {
		log.Printf("Failed to create model: %v", err)
		return
	}

	// Create text question
	textBlock, _ := types.NewContentBlock("text", []byte("What objects are visible in this image?"))

	// Create image block (using URL in real scenario)
	imageBlock, err := types.NewContentBlockFromURL(ctx, "image", "https://example.com/image.png")
	if err != nil {
		fmt.Printf("Note: Image URL fetch failed (expected in examples): %v\n", err)
		return
	}

	input, _ := types.NewMultimodalInput([]*types.ContentBlock{textBlock, imageBlock})
	output, err := model.Process(ctx, input)
	if err != nil {
		log.Printf("Processing failed: %v", err)
		return
	}

	fmt.Printf("Question: %s\n", string(textBlock.Data))
	if len(output.ContentBlocks) > 0 {
		fmt.Printf("Answer: %s\n", string(output.ContentBlocks[0].Data))
	}
}

func imageCaptioningExample(ctx context.Context) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Skipping: OPENAI_API_KEY not set")
		return
	}

	model, err := multimodal.NewMultimodalModel(ctx, "openai", multimodal.Config{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   apiKey,
		Timeout:  30 * time.Second,
	})
	if err != nil {
		log.Printf("Failed to create model: %v", err)
		return
	}

	// Create prompt for captioning
	textBlock, _ := types.NewContentBlock("text", []byte("Describe this image in detail."))
	imageBlock, err := types.NewContentBlockFromURL(ctx, "image", "https://example.com/image.png")
	if err != nil {
		fmt.Printf("Note: Image URL fetch failed (expected in examples): %v\n", err)
		return
	}

	input, _ := types.NewMultimodalInput([]*types.ContentBlock{textBlock, imageBlock})
	output, err := model.Process(ctx, input)
	if err != nil {
		log.Printf("Processing failed: %v", err)
		return
	}

	if len(output.ContentBlocks) > 0 {
		fmt.Printf("Caption: %s\n", string(output.ContentBlocks[0].Data))
	}
}

func multimodalReasoningExample(ctx context.Context) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Skipping: OPENAI_API_KEY not set")
		return
	}

	model, err := multimodal.NewMultimodalModel(ctx, "openai", multimodal.Config{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   apiKey,
		Timeout:  30 * time.Second,
	})
	if err != nil {
		log.Printf("Failed to create model: %v", err)
		return
	}

	// Create complex reasoning prompt with image
	textBlock, _ := types.NewContentBlock("text", []byte(
		"Analyze this image and explain: What is the main subject? " +
			"What is the context? What emotions or themes does it convey?",
	))
	imageBlock, err := types.NewContentBlockFromURL(ctx, "image", "https://example.com/image.png")
	if err != nil {
		fmt.Printf("Note: Image URL fetch failed (expected in examples): %v\n", err)
		return
	}

	input, _ := types.NewMultimodalInput([]*types.ContentBlock{textBlock, imageBlock})
	output, err := model.Process(ctx, input)
	if err != nil {
		log.Printf("Processing failed: %v", err)
		return
	}

	fmt.Printf("Reasoning Prompt: %s\n", string(textBlock.Data))
	if len(output.ContentBlocks) > 0 {
		fmt.Printf("Reasoning Result: %s\n", string(output.ContentBlocks[0].Data))
	}
}
