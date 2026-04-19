package llama

import (
	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/llm"
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
		return nil, core.Errorf(core.ErrInvalidInput, "llama: model is required")
	}

	backend, _ := config.GetOption[string](cfg, "backend")
	if backend == "" {
		backend = defaultBackend
	}

	baseURL, ok := backends[backend]
	if !ok {
		return nil, core.Errorf(core.ErrInvalidInput, "llama: unsupported backend %q, supported: together, fireworks, groq, sambanova, cerebras, ollama", backend)
	}

	// Only override BaseURL if not explicitly set.
	if cfg.BaseURL == "" {
		cfg.BaseURL = baseURL
	}

	// Delegate to the backend provider via the registry.
	return llm.New(backend, cfg)
}
