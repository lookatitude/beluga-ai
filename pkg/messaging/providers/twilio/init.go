package twilio

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/messaging"
	"github.com/lookatitude/beluga-ai/pkg/messaging/iface"
)

func init() {
	// Auto-register Twilio provider with the global registry (T065)
	messaging.GetRegistry().Register("twilio", func(ctx context.Context, config *messaging.Config) (iface.ConversationalBackend, error) {
		twilioConfig := NewTwilioConfig(config)
		return NewTwilioProvider(twilioConfig)
	})
}
