// docs/examples/config/main.go
package main

import (
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/config"
	// Ensure you have your config.yaml or environment variables set
)

func main() {
	// Load configuration from default paths (., $HOME/.beluga-ai, /etc/beluga-ai)
	// and environment variables (prefixed with BELUGA_)
	 err := config.LoadConfig()
	 if err != nil {
	 	 log.Fatalf("Failed to load configuration: %v", err)
	 }

	 fmt.Println("Configuration loaded successfully!")

	 // Access configuration values
	 fmt.Printf("OpenAI Model: %s\n", config.Cfg.LLMs.OpenAI.Model)
	 fmt.Printf("OpenAI API Key (masked): %s...\n", maskKey(config.Cfg.LLMs.OpenAI.APIKey))

	 fmt.Printf("Anthropic Model: %s\n", config.Cfg.LLMs.Anthropic.Model)
	 fmt.Printf("Anthropic API Key (masked): %s...\n", maskKey(config.Cfg.LLMs.Anthropic.APIKey))

	 fmt.Printf("Ollama Base URL: %s\n", config.Cfg.LLMs.Ollama.BaseURL)
	 fmt.Printf("Ollama Model: %s\n", config.Cfg.LLMs.Ollama.Model)

	 // Example accessing a potentially nested config (if added later)
	 // fmt.Printf("PGVector Connection String: %s\n", config.Cfg.RAG.PGVector.ConnectionString)
}

// Helper to mask API keys for printing
func maskKey(key string) string {
	 if len(key) > 4 {
	 	 return key[:4]
	 }
	 return ""
}

