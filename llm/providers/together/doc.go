// Package together provides the Together AI LLM provider for the Beluga AI
// framework.
//
// Together AI exposes an OpenAI-compatible API for running open-source models
// with optimized inference. This provider is a thin wrapper around the shared
// openaicompat package with Together's base URL.
//
// # Registration
//
// The provider registers itself as "together" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/together"
//
// # Usage
//
//	model, err := llm.New("together", config.ProviderConfig{
//	    Model:  "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
//	    APIKey: "...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Compare Llama and Mistral models"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the Together model path (defaults to "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo")
//   - APIKey: the Together API key
//   - BaseURL: optional, defaults to "https://api.together.xyz/v1"
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package together
