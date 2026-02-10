// Package azure provides the Azure OpenAI LLM provider for the Beluga AI
// framework.
//
// Azure OpenAI uses a different authentication scheme (api-key header) and URL
// structure (per-deployment endpoints with api-version query parameter) compared
// to the standard OpenAI API, but the request/response format is otherwise
// identical. This provider handles the Azure-specific authentication and URL
// rewriting transparently using the shared openaicompat package.
//
// # Registration
//
// The provider registers itself as "azure" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/azure"
//
// # Usage
//
//	model, err := llm.New("azure", config.ProviderConfig{
//	    Model:   "gpt-4o",
//	    APIKey:  "...",
//	    BaseURL: "https://myresource.openai.azure.com/openai/deployments/my-gpt4o",
//	    Options: map[string]any{"api_version": "2024-10-21"},
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Hello from Azure!"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the deployment model name (defaults to "gpt-4o")
//   - APIKey: the Azure OpenAI API key (required)
//   - BaseURL: the Azure deployment endpoint (required; format: https://{resource}.openai.azure.com/openai/deployments/{deployment})
//   - Options["api_version"]: the Azure API version (defaults to "2024-10-21")
//
// # Direct Construction
//
// Use [New] to create a ChatModel directly without going through the registry.
package azure
