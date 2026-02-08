// Package huggingface provides the HuggingFace Inference API LLM provider for
// the Beluga AI framework. HuggingFace exposes an OpenAI-compatible chat
// completions endpoint, so this provider is a thin wrapper around the shared
// openaicompat package.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/huggingface"
//
//	model, err := llm.New("huggingface", config.ProviderConfig{
//	    Model:  "meta-llama/Meta-Llama-3.1-70B-Instruct",
//	    APIKey: "hf_...",
//	})
package huggingface

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBaseURL = "https://api-inference.huggingface.co/v1"

func init() {
	llm.Register("huggingface", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new HuggingFace ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	return openaicompat.New(cfg)
}
