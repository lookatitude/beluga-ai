package amazon_nova

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
)

func init() {
	// Register Amazon Nova provider with the global registry
	s2s.GetRegistry().Register("amazon_nova", NewAmazonNovaProvider)
}
