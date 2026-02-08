// Package llama provides a Meta Llama model provider for the Beluga AI framework.
// Since Meta does not offer a direct API, this provider delegates to one of the
// available hosting providers that serve Llama models: Together, Fireworks, Groq,
// SambaNova, Cerebras, or Ollama.
//
// The backend can be selected via the "backend" option in ProviderConfig.Options.
// If not specified, it defaults to "together".
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/llama"
//
//	model, err := llm.New("llama", config.ProviderConfig{
//	    Model:  "meta-llama/Llama-3.3-70B-Instruct",
//	    APIKey: "...",
//	    Options: map[string]any{"backend": "together"},
//	})
package llama

import (
	"fmt"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBackend = "together"

// Supported backends and their default base URLs.
var backends = map[string]string{
	"together":  "https://api.together.xyz/v1",
	"fireworks": "https://api.fireworks.ai/inference/v1",
	"groq":      "https://api.groq.com/openai/v1",
	"sambanova": "https://api.sambanova.ai/v1",
	"cerebras":  "https://api.cerebras.ai/v1",
	"ollama":    "http://localhost:11434/v1",
}

func init() {
	llm.Register("llama", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new Llama ChatModel by delegating to a hosting provider.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("llama: model is required")
	}

	backend, _ := config.GetOption[string](cfg, "backend")
	if backend == "" {
		backend = defaultBackend
	}

	baseURL, ok := backends[backend]
	if !ok {
		return nil, fmt.Errorf("llama: unsupported backend %q, supported: together, fireworks, groq, sambanova, cerebras, ollama", backend)
	}

	// Only override BaseURL if not explicitly set.
	if cfg.BaseURL == "" {
		cfg.BaseURL = baseURL
	}

	// Delegate to the backend provider via the registry.
	return llm.New(backend, cfg)
}
