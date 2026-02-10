// Package xai provides the xAI Grok LLM provider for the Beluga AI framework.
//
// xAI exposes an OpenAI-compatible API for Grok models. This provider is a
// thin wrapper around the shared openaicompat package with xAI's base URL.
//
// # Registration
//
// The provider registers itself as "xai" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/xai"
//
// # Usage
//
//	model, err := llm.New("xai", config.ProviderConfig{
//	    Model:  "grok-3",
//	    APIKey: "xai-...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("What makes Grok unique?"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the Grok model name (defaults to "grok-3")
//   - APIKey: the xAI API key
//   - BaseURL: optional, defaults to "https://api.x.ai/v1"
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package xai
