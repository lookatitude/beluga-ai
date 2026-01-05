package openai_realtime

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
)

func init() {
	// Register OpenAI Realtime provider with the global registry
	s2s.GetRegistry().Register("openai_realtime", NewOpenAIRealtimeProvider)
}
