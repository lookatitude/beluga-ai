package azure

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
)

func init() {
	// Register Azure provider with the global registry
	tts.GetRegistry().Register("azure", NewAzureProvider)
}
