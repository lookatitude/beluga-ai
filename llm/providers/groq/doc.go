// Package groq provides the Groq LLM provider for the Beluga AI framework.
//
// Groq exposes an OpenAI-compatible API optimized for fast inference on
// custom LPU hardware. This provider is a thin wrapper around the shared
// openaicompat package with Groq's base URL.
//
// # Registration
//
// The provider registers itself as "groq" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/groq"
//
// # Usage
//
//	model, err := llm.New("groq", config.ProviderConfig{
//	    Model:  "llama-3.3-70b-versatile",
//	    APIKey: "gsk_...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Explain Groq's LPU architecture"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the Groq model name (e.g. "llama-3.3-70b-versatile", "mixtral-8x7b-32768")
//   - APIKey: the Groq API key
//   - BaseURL: optional, defaults to "https://api.groq.com/openai/v1"
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package groq
