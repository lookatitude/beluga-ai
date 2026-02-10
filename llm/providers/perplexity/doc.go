// Package perplexity provides the Perplexity LLM provider for the Beluga AI
// framework.
//
// Perplexity exposes an OpenAI-compatible API with models optimized for search
// and retrieval-augmented generation. This provider is a thin wrapper around
// the shared openaicompat package with Perplexity's base URL.
//
// # Registration
//
// The provider registers itself as "perplexity" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/perplexity"
//
// # Usage
//
//	model, err := llm.New("perplexity", config.ProviderConfig{
//	    Model:  "sonar-pro",
//	    APIKey: "pplx-...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("What happened in tech news today?"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the Perplexity model name (e.g. "sonar-pro", "sonar")
//   - APIKey: the Perplexity API key
//   - BaseURL: optional, defaults to "https://api.perplexity.ai"
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package perplexity
