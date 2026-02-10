// Package fireworks provides the Fireworks AI LLM provider for the Beluga AI
// framework.
//
// Fireworks AI exposes an OpenAI-compatible API optimized for fast inference
// of open-source models. This provider is a thin wrapper around the shared
// openaicompat package with Fireworks' base URL.
//
// # Registration
//
// The provider registers itself as "fireworks" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/fireworks"
//
// # Usage
//
//	model, err := llm.New("fireworks", config.ProviderConfig{
//	    Model:  "accounts/fireworks/models/llama-v3p1-70b-instruct",
//	    APIKey: "fw_...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Write a function to sort a list"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the Fireworks model path (defaults to "accounts/fireworks/models/llama-v3p1-70b-instruct")
//   - APIKey: the Fireworks API key
//   - BaseURL: optional, defaults to "https://api.fireworks.ai/inference/v1"
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package fireworks
