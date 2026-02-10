// Package openai provides the OpenAI LLM provider for the Beluga AI framework.
//
// It implements the [llm.ChatModel] interface using the openai-go SDK via the
// shared openaicompat package. This provider supports all OpenAI chat models
// including GPT-4o, GPT-4, and GPT-3.5 Turbo, with full support for
// streaming, tool calling, structured output, and multimodal inputs.
//
// # Registration
//
// The provider registers itself as "openai" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
//
// # Usage
//
//	model, err := llm.New("openai", config.ProviderConfig{
//	    Model:  "gpt-4o",
//	    APIKey: "sk-...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Hello, world!"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the OpenAI model name (e.g. "gpt-4o", "gpt-4o-mini")
//   - APIKey: the OpenAI API key
//   - BaseURL: optional, defaults to "https://api.openai.com/v1"
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package openai
