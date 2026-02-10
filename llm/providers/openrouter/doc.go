// Package openrouter provides the OpenRouter LLM provider for the Beluga AI
// framework.
//
// OpenRouter exposes an OpenAI-compatible API that routes requests to many
// different model providers, enabling access to hundreds of models through a
// single API key. This provider is a thin wrapper around the shared openaicompat
// package with OpenRouter's base URL.
//
// # Registration
//
// The provider registers itself as "openrouter" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/openrouter"
//
// # Usage
//
//	model, err := llm.New("openrouter", config.ProviderConfig{
//	    Model:  "anthropic/claude-sonnet-4-5-20250929",
//	    APIKey: "sk-or-...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Compare GPT-4 and Claude"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the model path in provider/model format (e.g. "anthropic/claude-sonnet-4-5-20250929", "openai/gpt-4o")
//   - APIKey: the OpenRouter API key
//   - BaseURL: optional, defaults to "https://openrouter.ai/api/v1"
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package openrouter
