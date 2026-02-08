// Package qwen provides the Alibaba Qwen LLM provider for the Beluga AI framework.
// Qwen exposes an OpenAI-compatible API, so this provider is a thin wrapper
// around the shared openaicompat package with Qwen's base URL.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/qwen"
//
//	model, err := llm.New("qwen", config.ProviderConfig{
//	    Model:  "qwen-plus",
//	    APIKey: "sk-...",
//	})
package qwen

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"

func init() {
	llm.Register("qwen", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new Qwen ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	return openaicompat.New(cfg)
}
