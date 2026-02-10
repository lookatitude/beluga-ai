// Package qwen provides the Alibaba Qwen LLM provider for the Beluga AI
// framework.
//
// Qwen exposes an OpenAI-compatible API through Alibaba's DashScope platform,
// so this provider is a thin wrapper around the shared openaicompat package
// with Qwen's base URL.
//
// # Registration
//
// The provider registers itself as "qwen" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/qwen"
//
// # Usage
//
//	model, err := llm.New("qwen", config.ProviderConfig{
//	    Model:  "qwen-plus",
//	    APIKey: "sk-...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Explain the Qwen model family"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the Qwen model name (e.g. "qwen-plus", "qwen-turbo", "qwen-max")
//   - APIKey: the DashScope API key
//   - BaseURL: optional, defaults to "https://dashscope.aliyuncs.com/compatible-mode/v1"
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package qwen
