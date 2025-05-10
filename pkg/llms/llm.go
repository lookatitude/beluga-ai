package llms

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// LLM defines the interface for a Large Language Model.
// It provides methods for invoking the model and potentially for streaming responses.
type LLM interface {
	// Invoke sends a single request to the LLM and gets a single response.
	// The input prompt can be a simple string or a more complex structure depending on the model.
	// callOptions can be used to override default model parameters like temperature, max_tokens, etc.
	Invoke(ctx context.Context, prompt string, callOptions ...schema.LLMOption) (string, error)

	// TODO: Add a Generate method for more complex interactions, possibly returning multiple choices or structured output.
	// Generate(ctx context.Context, prompts []string, callOptions ...schema.LLMOption) ([]schema.Generation, error)

	// TODO: Add a Stream method for streaming responses.
	// Stream(ctx context.Context, prompt string, callOptions ...schema.LLMOption) (<-chan schema.LLMResponseChunk, error)

	// GetModelName returns the specific model name being used by this LLM instance.
	GetModelName() string

	// GetProviderName returns the name of the LLM provider (e.g., "openai", "anthropic").
	GetProviderName() string
}

// LLMFactory defines the interface for creating LLM instances.
// This allows for different LLM providers to be instantiated based on configuration.
type LLMFactory interface {
	CreateLLM(ctx context.Context, config schema.LLMProviderConfig) (LLM, error)
}

// ConcreteLLMFactory is a basic implementation of LLMFactory.
// It will need to be populated with specific provider constructors.
// For now, it serves as a placeholder and will be expanded as providers are added.
type ConcreteLLMFactory struct {
	// In a real implementation, this might hold API keys or other shared resources
	// or a map of provider-specific factory functions.
	// For now, we keep it simple and the CreateLLM method will have a switch statement.
}

// NewConcreteLLMFactory creates a new ConcreteLLMFactory.
func NewConcreteLLMFactory() *ConcreteLLMFactory {
	return &ConcreteLLMFactory{}
}

// CreateLLM creates an LLM instance based on the provider specified in the config.
// This is a simplified version; a more robust factory might use registration of provider-specific factories.
func (f *ConcreteLLMFactory) CreateLLM(ctx context.Context, config schema.LLMProviderConfig) (LLM, error) {
	// Provider-specific logic will be added here.
	// Example for OpenAI will be in its own package (e.g., pkg/llms/openai)
	// and this factory would call out to it.
	// For now, this is a placeholder to be filled as providers are implemented.
	// This factory will eventually call constructors like openai.NewOpenAI_LLM(config)

	// This function will be updated in subsequent steps when specific providers like OpenAI are implemented.
	// The actual instantiation logic will be delegated to provider-specific packages.
	// For example, if config.Provider == "openai", it would call a function from the openai package.
	return nil, fmt.Errorf("LLM provider 	%s	 not yet supported by ConcreteLLMFactory", config.Provider)
}

