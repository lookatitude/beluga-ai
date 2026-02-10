// Package anthropic provides the Anthropic (Claude) LLM provider for the
// Beluga AI framework.
//
// It implements the [llm.ChatModel] interface using the anthropic-sdk-go SDK,
// with native support for the Anthropic Messages API including streaming,
// tool use, vision (image inputs), and prompt caching.
//
// # Registration
//
// The provider registers itself as "anthropic" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
//
// # Usage
//
//	model, err := llm.New("anthropic", config.ProviderConfig{
//	    Model:  "claude-sonnet-4-5-20250929",
//	    APIKey: "sk-ant-...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Explain quantum computing"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the Anthropic model name (required; e.g. "claude-sonnet-4-5-20250929", "claude-haiku-4-5-20251001")
//   - APIKey: the Anthropic API key
//   - BaseURL: optional, defaults to Anthropic's API endpoint
//   - Timeout: optional request timeout
//
// # Key Types
//
//   - [Model]: the ChatModel implementation with Generate, Stream, BindTools, and ModelID methods
//   - [New]: constructor that creates a Model from a [config.ProviderConfig]
//
// # Implementation Notes
//
// Unlike the OpenAI-compatible providers, this package implements the full
// Anthropic Messages API natively. System messages are extracted and passed
// as the dedicated system parameter. Tool use follows the Anthropic tool_use
// content block format. The default max tokens is 4096 when not specified.
package anthropic
