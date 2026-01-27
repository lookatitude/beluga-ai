package openai

import (
	"github.com/lookatitude/beluga-ai/pkg/stt"
)

func init() {
	// Register OpenAI provider with the global registry
	stt.GetRegistry().Register("openai", NewOpenAIProvider)
}
