package openai

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
)

func init() {
	// Register OpenAI provider with the global registry
	tts.GetRegistry().Register("openai", NewOpenAIProvider)
}
