// Package bifrost provides a ChatModel backed by a Bifrost gateway.
// Bifrost is an OpenAI-compatible proxy that routes requests to multiple
// LLM providers with load balancing and failover. This provider is a thin
// wrapper around the shared openaicompat package.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/bifrost"
//
//	model, err := llm.New("bifrost", config.ProviderConfig{
//	    Model:   "gpt-4o",
//	    APIKey:  "sk-...",
//	    BaseURL: "http://localhost:8080/v1",
//	})
package bifrost

import (
	"fmt"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

func init() {
	llm.Register("bifrost", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new Bifrost ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("bifrost: base_url is required")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("bifrost: model is required")
	}
	return openaicompat.New(cfg)
}
