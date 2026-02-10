// Package google provides the Google Gemini LLM provider for the Beluga AI
// framework.
//
// It implements the [llm.ChatModel] interface using the google.golang.org/genai
// SDK, with native support for the Gemini API including streaming, function
// calling, vision (image and file inputs), and system instructions.
//
// # Registration
//
// The provider registers itself as "google" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/google"
//
// # Usage
//
//	model, err := llm.New("google", config.ProviderConfig{
//	    Model:  "gemini-2.5-flash",
//	    APIKey: "...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Summarize this document"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the Gemini model name (required; e.g. "gemini-2.5-flash", "gemini-2.5-pro")
//   - APIKey: the Google AI API key
//   - BaseURL: optional, overrides the default Gemini API endpoint
//
// # Key Types
//
//   - [Model]: the ChatModel implementation
//   - [New]: constructor from [config.ProviderConfig]
//   - [NewWithHTTPClient]: constructor accepting a custom *http.Client for testing
//
// # Implementation Notes
//
// This package implements the Gemini API natively rather than using the
// OpenAI compatibility layer. System messages are passed as system
// instructions. Tool calling uses the Gemini function calling format with
// FunctionDeclaration and FunctionResponse parts. Image inputs support
// both inline data (base64) and file URIs.
package google
