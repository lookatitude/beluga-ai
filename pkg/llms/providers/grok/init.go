package grok

import "github.com/lookatitude/beluga-ai/pkg/llms"

func init() {
	// Register Grok provider with the global registry
	llms.GetRegistry().Register("grok", NewGrokProviderFactory())
}
