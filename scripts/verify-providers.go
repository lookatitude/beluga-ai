//go:build ignore
// +build ignore

// This script verifies that all new providers are properly registered.
// Run with: go run scripts/verify-providers.go
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	// Import providers to trigger their init() functions
	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/google_multimodal"
	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/openai_multimodal"
	_ "github.com/lookatitude/beluga-ai/pkg/llms/providers/gemini"
	_ "github.com/lookatitude/beluga-ai/pkg/llms/providers/grok"
	_ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/qdrant"
	_ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/weaviate"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func main() {
	fmt.Println("Verifying provider registrations...")
	fmt.Println()

	// Verify LLM providers
	fmt.Println("LLM Providers:")
	llmRegistry := llms.GetRegistry()
	llmProviders := llmRegistry.ListProviders()
	for _, name := range llmProviders {
		registered := llmRegistry.IsRegistered(name)
		status := "✓"
		if !registered {
			status = "✗"
		}
		fmt.Printf("  %s %s\n", status, name)
	}

	// Check for new providers
	newLLMProviders := []string{"grok", "gemini"}
	for _, name := range newLLMProviders {
		if llmRegistry.IsRegistered(name) {
			fmt.Printf("  ✓ NEW: %s\n", name)
		} else {
			fmt.Printf("  ✗ MISSING: %s\n", name)
			os.Exit(1)
		}
	}

	fmt.Println()

	// Verify Embedding providers
	fmt.Println("Embedding Providers:")
	embeddingRegistry := embeddings.GetRegistry()
	embeddingProviders := embeddingRegistry.ListProviders()
	for _, name := range embeddingProviders {
		fmt.Printf("  ✓ %s\n", name)
	}

	// Check for new providers
	newEmbeddingProviders := []string{"openai_multimodal", "google_multimodal"}
	for _, name := range newEmbeddingProviders {
		if embeddingRegistry.IsRegistered(name) {
			fmt.Printf("  ✓ NEW: %s\n", name)
		} else {
			fmt.Printf("  ✗ MISSING: %s\n", name)
			os.Exit(1)
		}
	}

	fmt.Println()

	// Verify Vector Store providers
	fmt.Println("Vector Store Providers:")
	// Note: vectorstores uses a factory pattern
	vsRegistry := vectorstores.GetRegistry()

	// Check for new providers by attempting to create (will fail on config, but proves registration)
	newVSProviders := []string{"qdrant", "weaviate"}
	for _, name := range newVSProviders {
		config := vectorstores.NewDefaultConfig()
		config.ProviderConfig = map[string]any{
			"collection_name": "test",
			"class_name":      "Test",
		}
		// The error will tell us if it's registered or not
		_, err := vsRegistry.Create(context.Background(), name, *config)
		if err != nil {
			// If error mentions "not found" or "unknown provider", it's not registered
			errStr := err.Error()
			if contains(errStr, "not found") || contains(errStr, "unknown provider") {
				fmt.Printf("  ✗ MISSING: %s\n", name)
				os.Exit(1)
			} else {
				// Any other error means it's registered (just config issue)
				fmt.Printf("  ✓ NEW: %s (registered, config error expected)\n", name)
			}
		}
	}

	fmt.Println()
	fmt.Println("✓ All provider verifications complete!")
}
