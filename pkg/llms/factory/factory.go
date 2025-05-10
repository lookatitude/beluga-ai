package factory

import (
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/mock"
	"github.com/lookatitude/beluga-ai/pkg/llms/openai"
	// "github.com/lookatitude/beluga-ai/pkg/schema" // Removed unused import
)

// LLMProviderFactory defines the interface for an LLM provider factory.
type LLMProviderFactory interface {
	GetProvider(name string) (llms.LLM, error)
	RegisterProvider(name string, provider llms.LLM) // Optional: for manual registration
}

// llmProviderFactoryImpl is the concrete implementation of LLMProviderFactory.
type llmProviderFactoryImpl struct {
	configProvider config.Provider
	providers      map[string]llms.LLM
}

// NewLLMProviderFactory creates a new LLM provider factory.
// It initializes providers based on the application configuration.
func NewLLMProviderFactory(configProvider config.Provider) (LLMProviderFactory, error) {
	factory := &llmProviderFactoryImpl{
		configProvider: configProvider,
		providers:      make(map[string]llms.LLM),
	}

	llmConfigs, err := configProvider.GetLLMProvidersConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM providers config: %w", err)
	}

	for _, providerConfig := range llmConfigs { // providerConfig is schema.LLMProviderConfig
		var providerInstance llms.LLM
		switch providerConfig.Provider {
		case "openai":
			// Use config.OpenAILLMConfig from the config package
			cfg := config.OpenAILLMConfig{
				Model:       providerConfig.ModelName,
				APIKey:      providerConfig.APIKey,
			}
			if temp, ok := providerConfig.DefaultCallOptions["temperature"].(float64); ok {
				cfg.Temperature = temp
			}
			if maxTokens, ok := providerConfig.DefaultCallOptions["max_tokens"].(float64); ok { // Viper might unmarshal numbers as float64
				cfg.MaxTokens = int(maxTokens)
			} else if maxTokensInt, ok := providerConfig.DefaultCallOptions["max_tokens"].(int); ok {
			    cfg.MaxTokens = maxTokensInt
            }

            if topP, ok := providerConfig.DefaultCallOptions["top_p"].(float64); ok {
                cfg.TopP = topP
            }
            if freqPenalty, ok := providerConfig.DefaultCallOptions["frequency_penalty"].(float64); ok {
                cfg.FrequencyPenalty = freqPenalty
            }
            if presPenalty, ok := providerConfig.DefaultCallOptions["presence_penalty"].(float64); ok {
                cfg.PresencePenalty = presPenalty
            }
            if stop, ok := providerConfig.DefaultCallOptions["stop"].([]string); ok {
                cfg.Stop = stop
            } else if stopInterface, ok := providerConfig.DefaultCallOptions["stop"].([]interface{}); ok {
                for _, s := range stopInterface {
                    if str, okStr := s.(string); okStr {
                        cfg.Stop = append(cfg.Stop, str)
                    }
                }
            }
            if streaming, ok := providerConfig.DefaultCallOptions["streaming"].(bool); ok {
                cfg.Streaming = streaming
            }
            if apiVersion, ok := providerConfig.ProviderSpecific["api_version"].(string); ok {
                cfg.APIVersion = apiVersion
            }
            if apiEndpoint, ok := providerConfig.ProviderSpecific["api_endpoint"].(string); ok {
                cfg.APIEndpoint = apiEndpoint
            }
             if timeout, ok := providerConfig.ProviderSpecific["timeout_seconds"].(float64); ok {
                cfg.Timeout = int(timeout)
            } else if timeoutInt, ok := providerConfig.ProviderSpecific["timeout_seconds"].(int); ok {
                cfg.Timeout = timeoutInt
            }


			providerInstance, err = openai.NewOpenAILLM(cfg) // Ensure NewOpenAILLM exists and has this signature
			if err != nil {
				return nil, fmt.Errorf("failed to create openai llm 	%s	: %w", providerConfig.Name, err)
			}
		case "mock":
			// Use config.MockLLMConfig from the config package
			mockCfg := config.MockLLMConfig{
				ModelName: providerConfig.ModelName,
			}
            if expectedError, ok := providerConfig.ProviderSpecific["expected_error"].(string); ok {
                mockCfg.ExpectedError = expectedError
            }
            if responsesInterface, ok := providerConfig.ProviderSpecific["responses"].([]interface{}); ok {
                for _, r := range responsesInterface {
                    if str, okStr := r.(string); okStr {
                        mockCfg.Responses = append(mockCfg.Responses, str)
                    }
                }
            }

			providerInstance, err = mock.NewMockLLM(mockCfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create mock llm 	%s	: %w", providerConfig.Name, err)
			}
		default:
			return nil, fmt.Errorf("unsupported LLM provider type: %s for provider %s", providerConfig.Provider, providerConfig.Name)
		}
		factory.providers[providerConfig.Name] = providerInstance
	}

	return factory, nil
}

// GetProvider returns an LLM provider by its name.
func (f *llmProviderFactoryImpl) GetProvider(name string) (llms.LLM, error) {
	provider, ok := f.providers[name]
	if !ok {
		return nil, fmt.Errorf("LLM provider with name 	%s	 not found or not configured", name)
	}
	return provider, nil
}

// RegisterProvider allows manual registration of an LLM provider.
func (f *llmProviderFactoryImpl) RegisterProvider(name string, provider llms.LLM) {
	f.providers[name] = provider
}

