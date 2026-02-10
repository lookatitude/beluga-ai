// Package deepseek provides the DeepSeek LLM provider for the Beluga AI
// framework.
//
// DeepSeek exposes an OpenAI-compatible API, so this provider is a thin
// wrapper around the shared openaicompat package with DeepSeek's base URL.
// It supports DeepSeek's chat and reasoning models.
//
// # Registration
//
// The provider registers itself as "deepseek" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/deepseek"
//
// # Usage
//
//	model, err := llm.New("deepseek", config.ProviderConfig{
//	    Model:  "deepseek-chat",
//	    APIKey: "sk-...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Solve this math problem"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the DeepSeek model name (defaults to "deepseek-chat")
//   - APIKey: the DeepSeek API key
//   - BaseURL: optional, defaults to "https://api.deepseek.com/v1"
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package deepseek
