// Package llama provides a Meta Llama model provider for the Beluga AI
// framework.
//
// Since Meta does not offer a direct API for Llama models, this provider
// delegates to one of the available hosting backends that serve Llama models:
// Together, Fireworks, Groq, SambaNova, Cerebras, or Ollama. The backend
// is selected via the "backend" option in [config.ProviderConfig].Options.
//
// # Registration
//
// The provider registers itself as "llama" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/llama"
//
// # Usage
//
//	model, err := llm.New("llama", config.ProviderConfig{
//	    Model:   "meta-llama/Llama-3.3-70B-Instruct",
//	    APIKey:  "...",
//	    Options: map[string]any{"backend": "together"},
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Explain the Llama architecture"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the Llama model name (required; e.g. "meta-llama/Llama-3.3-70B-Instruct")
//   - APIKey: the API key for the chosen backend
//   - BaseURL: optional, overrides the backend's default URL
//   - Options["backend"]: hosting backend to use (defaults to "together"; supported: "together", "fireworks", "groq", "sambanova", "cerebras", "ollama")
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
// The appropriate backend provider must be imported for delegation to work.
package llama
