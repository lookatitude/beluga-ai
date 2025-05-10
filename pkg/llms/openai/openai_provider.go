package openai

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/sashabaranov/go-openai"
)

// OpenAI_LLM is an implementation of the llms.LLM interface for OpenAI models.
type OpenAI_LLM struct {
	client *openai.Client
	config schema.LLMProviderConfig
}

// NewOpenAI_LLM creates a new OpenAI_LLM instance.
func NewOpenAI_LLM(config schema.LLMProviderConfig) (*OpenAI_LLM, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	openaiConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		openaiConfig.BaseURL = config.BaseURL
	}
	// TODO: Potentially add other openai.ClientConfig options from config.ProviderSpecific
	// like OrganizationID, HTTPClient, etc.

	client := openai.NewClientWithConfig(openaiConfig)

	return &OpenAI_LLM{
		client: client,
		config: config,
	}, nil
}

// Invoke sends a request to the OpenAI LLM.
func (o *OpenAI_LLM) Invoke(ctx context.Context, prompt string, callOptions ...schema.LLMOption) (string, error) {
	// Apply call options to override defaults from config
	// For now, we directly use config.DefaultCallOptions and assume they are compatible
	// A more robust way would be to merge and convert them to openai.ChatCompletionRequest fields.

	// For simplicity, we use ChatCompletion API. For pure completion, Completion API could be used.
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	request := openai.ChatCompletionRequest{
		Model:    o.config.ModelName,
		Messages: messages,
	}

	// Apply default call options from config if any
	if o.config.DefaultCallOptions != nil {
		if temp, ok := o.config.DefaultCallOptions["temperature"].(float32); ok {
			request.Temperature = temp
		}
		if maxTokens, ok := o.config.DefaultCallOptions["max_tokens"].(int); ok {
			request.MaxTokens = maxTokens
		}
		if topP, ok := o.config.DefaultCallOptions["top_p"].(float32); ok {
			request.TopP = topP
		}
		// TODO: Add more option conversions (e.g., presence_penalty, frequency_penalty, stop_sequences)
	}

	// Apply runtime callOptions (these would override the defaults)
	// This part needs a proper schema.LLMOption definition and parsing logic.
	// For now, we skip applying runtime callOptions for simplicity in this initial implementation.
	// for _, opt := range callOptions {
	// 	 opt.Apply(&request) // Assuming LLMOption has an Apply method
	// }

	resp, err := o.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return "", fmt.Errorf("OpenAI chat completion error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("OpenAI returned no choices")
	}

	return resp.Choices[0].Message.Content, nil
}

// GetModelName returns the model name.
func (o *OpenAI_LLM) GetModelName() string {
	return o.config.ModelName
}

// GetProviderName returns the provider name.
func (o *OpenAI_LLM) GetProviderName() string {
	return "openai"
}

// Ensure OpenAI_LLM implements the llms.LLM interface.
var _ llms.LLM = (*OpenAI_LLM)(nil)

