// Package cohere provides the Cohere LLM provider for the Beluga AI framework.
//
// It implements the [llm.ChatModel] interface using the official Cohere Go SDK
// (v2), with native support for the Cohere Chat API including streaming, tool
// use, and the unique Cohere message format.
//
// Cohere uses a different message structure than OpenAI: the last user message
// becomes the "message" field, system messages go into the "preamble", and all
// prior messages become "chat_history". This provider handles the mapping
// transparently.
//
// # Registration
//
// The provider registers itself as "cohere" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/cohere"
//
// # Usage
//
//	model, err := llm.New("cohere", config.ProviderConfig{
//	    Model:  "command-r-plus",
//	    APIKey: "...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Explain RAG in three sentences"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the Cohere model name (defaults to "command-r-plus")
//   - APIKey: the Cohere API key (required)
//   - BaseURL: optional, overrides the default Cohere API endpoint
//
// # Key Types
//
//   - [Model]: the ChatModel implementation using the Cohere SDK
//   - [New]: constructor from [config.ProviderConfig]
//
// # Implementation Notes
//
// This package uses the Cohere Go SDK directly rather than the OpenAI
// compatibility layer. Tool definitions are converted to Cohere's
// ParameterDefinitions format. Streaming uses Cohere's event-based
// stream with TextGeneration, ToolCallsGeneration, and StreamEnd events.
package cohere
