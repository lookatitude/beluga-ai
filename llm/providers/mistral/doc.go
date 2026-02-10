// Package mistral provides the Mistral AI LLM provider for the Beluga AI
// framework.
//
// It implements the [llm.ChatModel] interface using the official Mistral Go
// SDK, with native support for Mistral's chat API including streaming, tool
// use, and JSON output mode.
//
// # Registration
//
// The provider registers itself as "mistral" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/mistral"
//
// # Usage
//
//	model, err := llm.New("mistral", config.ProviderConfig{
//	    Model:  "mistral-large-latest",
//	    APIKey: "...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Write a haiku about Go"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the Mistral model name (defaults to "mistral-large-latest")
//   - APIKey: the Mistral API key (required)
//   - BaseURL: optional, defaults to "https://api.mistral.ai"
//   - Timeout: optional request timeout (defaults to 30s)
//
// # Key Types
//
//   - [Model]: the ChatModel implementation using the Mistral SDK
//   - [New]: constructor from [config.ProviderConfig]
//
// # Implementation Notes
//
// This package uses the Mistral Go SDK directly rather than the OpenAI
// compatibility layer. Streaming is channel-based in the Mistral SDK and
// is adapted to the iter.Seq2 pattern. Tool definitions are converted to
// the Mistral function calling format.
package mistral
