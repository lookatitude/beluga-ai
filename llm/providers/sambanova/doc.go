// Package sambanova provides the SambaNova LLM provider for the Beluga AI
// framework.
//
// SambaNova exposes an OpenAI-compatible API running on custom RDU hardware
// optimized for high-throughput inference. This provider is a thin wrapper
// around the shared openaicompat package with SambaNova's base URL.
//
// # Registration
//
// The provider registers itself as "sambanova" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/sambanova"
//
// # Usage
//
//	model, err := llm.New("sambanova", config.ProviderConfig{
//	    Model:  "Meta-Llama-3.3-70B-Instruct",
//	    APIKey: "sn-...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Write a parallel algorithm"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the SambaNova model name (e.g. "Meta-Llama-3.3-70B-Instruct")
//   - APIKey: the SambaNova API key
//   - BaseURL: optional, defaults to "https://api.sambanova.ai/v1"
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package sambanova
