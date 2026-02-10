// Package cerebras provides the Cerebras LLM provider for the Beluga AI
// framework.
//
// Cerebras exposes an OpenAI-compatible API running on Wafer-Scale Engine
// hardware optimized for fast inference. This provider is a thin wrapper
// around the shared openaicompat package with Cerebras' base URL.
//
// # Registration
//
// The provider registers itself as "cerebras" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/cerebras"
//
// # Usage
//
//	model, err := llm.New("cerebras", config.ProviderConfig{
//	    Model:  "llama-3.3-70b",
//	    APIKey: "csk-...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Explain wafer-scale computing"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the Cerebras model name (e.g. "llama-3.3-70b")
//   - APIKey: the Cerebras API key
//   - BaseURL: optional, defaults to "https://api.cerebras.ai/v1"
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package cerebras
