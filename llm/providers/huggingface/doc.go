// Package huggingface provides the HuggingFace Inference API LLM provider for
// the Beluga AI framework.
//
// HuggingFace exposes an OpenAI-compatible chat completions endpoint through
// its Inference API, so this provider is a thin wrapper around the shared
// openaicompat package. It supports any model hosted on HuggingFace's
// inference infrastructure.
//
// # Registration
//
// The provider registers itself as "huggingface" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/huggingface"
//
// # Usage
//
//	model, err := llm.New("huggingface", config.ProviderConfig{
//	    Model:  "meta-llama/Meta-Llama-3.1-70B-Instruct",
//	    APIKey: "hf_...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("What is transfer learning?"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the HuggingFace model ID (e.g. "meta-llama/Meta-Llama-3.1-70B-Instruct")
//   - APIKey: the HuggingFace API token
//   - BaseURL: optional, defaults to "https://api-inference.huggingface.co/v1"
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package huggingface
