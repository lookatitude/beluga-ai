package twilio

import (
	"github.com/lookatitude/beluga-ai/pkg/voicebackend"
)

func init() {
	// Register Twilio provider with the global registry
	provider := NewTwilioProvider()
	voicebackend.GetRegistry().Register("twilio", provider.CreateBackend)
}
