// Package bedrock provides the AWS Bedrock LLM provider for the Beluga AI
// framework.
//
// It implements the [llm.ChatModel] interface using the AWS SDK v2 Bedrock
// Runtime Converse API, supporting all models available through Amazon
// Bedrock including Anthropic Claude, Meta Llama, Mistral, and Amazon Titan.
//
// # Registration
//
// The provider registers itself as "bedrock" via init(). Import the package
// for side effects to make it available through the llm registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/bedrock"
//
// # Usage
//
//	model, err := llm.New("bedrock", config.ProviderConfig{
//	    Model:   "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
//	    Options: map[string]any{"region": "us-east-1"},
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := model.Generate(ctx, []schema.Message{
//	    schema.NewHumanMessage("Analyze this data"),
//	})
//
// # Configuration
//
// The following [config.ProviderConfig] fields are used:
//
//   - Model: the Bedrock model ID (required; e.g. "us.anthropic.claude-sonnet-4-5-20250929-v1:0")
//   - APIKey: optional AWS access key ID (uses default credentials if unset)
//   - BaseURL: optional custom Bedrock endpoint
//   - Options["region"]: AWS region (defaults to "us-east-1")
//   - Options["secret_key"]: AWS secret access key (used with APIKey)
//
// # Key Types
//
//   - [Model]: the ChatModel implementation using the Bedrock Converse API
//   - [ConverseAPI]: interface for the subset of bedrockruntime.Client methods used, enabling mock injection for tests
//   - [New]: constructor from [config.ProviderConfig]
//   - [NewWithClient]: constructor accepting a custom [ConverseAPI] implementation for testing
//
// # Implementation Notes
//
// This package uses the Bedrock Converse API (not InvokeModel), which provides
// a unified interface across all Bedrock-hosted models. Authentication uses
// the standard AWS credential chain. System messages are passed as dedicated
// system content blocks. Tool use follows the Bedrock tool specification format.
package bedrock
