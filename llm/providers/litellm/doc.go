// Package litellm provides a ChatModel backed by a LiteLLM gateway for the
// Beluga AI framework.
//
// LiteLLM (https://litellm.ai) is a proxy that exposes an OpenAI-compatible
// API in front of 100+ LLM providers. This provider is a thin wrapper around
// the shared openaicompat package pointed at the LiteLLM proxy endpoint.
//
// # Registration
//
// The provider registers itself as "litellm" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/litellm"
//
// # Usage
//
//	model, err := llm.New("litellm", config.ProviderConfig{
//	    Model:   "gpt-4o",
//	    APIKey:  "sk-...",
//	    BaseURL: "http://localhost:4000/v1",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Hello from LiteLLM!"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the model name as configured in LiteLLM (defaults to "gpt-4o")
//   - APIKey: the LiteLLM proxy API key
//   - BaseURL: the LiteLLM proxy URL (defaults to "http://localhost:4000/v1")
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package litellm
