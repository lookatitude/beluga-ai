package azure

import (
	"github.com/lookatitude/beluga-ai/pkg/stt"
)

func init() {
	// Register Azure provider with the global registry
	stt.GetRegistry().Register("azure", NewAzureProvider)
}
