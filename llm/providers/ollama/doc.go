// Package ollama provides the Ollama LLM provider for the Beluga AI framework.
//
// Ollama exposes an OpenAI-compatible API, so this provider is a thin wrapper
// around the shared openaicompat package pointed at Ollama's local endpoint.
// It supports all models available through Ollama including Llama, Mistral,
// Phi, Gemma, and other open-source models.
//
// # Registration
//
// The provider registers itself as "ollama" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/ollama"
//
// # Usage
//
//	model, err := llm.New("ollama", config.ProviderConfig{
//	    Model: "llama3.2",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Hello!"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the Ollama model name (e.g. "llama3.2", "mistral", "phi3")
//   - BaseURL: optional, defaults to "http://localhost:11434/v1"
//   - APIKey: optional, defaults to "ollama" (Ollama does not require authentication)
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package ollama
