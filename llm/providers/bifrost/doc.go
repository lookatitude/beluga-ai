// Package bifrost provides a ChatModel backed by a Bifrost gateway for the
// Beluga AI framework.
//
// Bifrost is an OpenAI-compatible proxy that routes requests to multiple LLM
// providers with load balancing and failover. This provider is a thin wrapper
// around the shared openaicompat package pointed at a Bifrost proxy endpoint.
//
// # Registration
//
// The provider registers itself as "bifrost" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/bifrost"
//
// # Usage
//
//	model, err := llm.New("bifrost", config.ProviderConfig{
//	    Model:   "gpt-4o",
//	    APIKey:  "sk-...",
//	    BaseURL: "http://localhost:8080/v1",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Hello from Bifrost!"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the model name (required)
//   - APIKey: the API key for the Bifrost proxy
//   - BaseURL: the Bifrost proxy URL (required; no default)
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package bifrost
