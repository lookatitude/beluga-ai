package openai

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/registry"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

const (
	DefaultOpenAIEmbeddingsModel = "text-embedding-ada-002"
	Ada002Dimension              = 1536
	ProviderName                 = "openai"
)

func init() {
	registry.RegisterEmbedder(ProviderName, func(generalConfig config.Provider, instanceConfig schema.EmbeddingProviderConfig) (iface.Embedder, error) {
		cfg := config.OpenAIEmbedderConfig{
			Model:   instanceConfig.ModelName,
			APIKey:  instanceConfig.APIKey,
		}
		if instanceConfig.ProviderSpecific != nil {
			if baseURL, ok := instanceConfig.ProviderSpecific["base_url"].(string); ok {
				cfg.APIEndpoint = baseURL
			}
			if apiVersion, ok := instanceConfig.ProviderSpecific["api_version"].(string); ok {
				cfg.APIVersion = apiVersion
			}
			if timeoutVal, ok := instanceConfig.ProviderSpecific["timeout_seconds"]; ok {
				if floatVal, isFloat := timeoutVal.(float64); isFloat {
					cfg.Timeout = int(floatVal)
				} else if intVal, isInt := timeoutVal.(int); isInt {
					cfg.Timeout = intVal
				}
			}
		}
		return NewOpenAIEmbedder(cfg)
	})
}

type OpenAIEmbedder struct {
	config config.OpenAIEmbedderConfig
}

func NewOpenAIEmbedder(cfg config.OpenAIEmbedderConfig) (*OpenAIEmbedder, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required and was not found in config")
	}
	if cfg.Model == "" {
		cfg.Model = DefaultOpenAIEmbeddingsModel
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 // Default timeout 30 seconds
	}

	return &OpenAIEmbedder{
		config: cfg,
	}, nil
}

func (e *OpenAIEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	fmt.Printf("Simulating OpenAI API call for %d documents with model %s\n", len(texts), e.config.Model)
	time.Sleep(100 * time.Millisecond)
	results := make([][]float32, len(texts))
	dim, _ := e.GetDimension(ctx)
	for i := range texts {
		dummyEmbedding := make([]float32, dim)
		for j := 0; j < dim; j++ {
			dummyEmbedding[j] = float32(i+1) * 0.1 * float32(j+1)
		}
		results[i] = dummyEmbedding
	}
	return results, nil
}

func (e *OpenAIEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("cannot embed empty query text")
	}
	fmt.Printf("Simulating OpenAI API call for query 	%s	 with model %s\n", text, e.config.Model)
	time.Sleep(50 * time.Millisecond)
	dim, _ := e.GetDimension(ctx)
	dummyEmbedding := make([]float32, dim)
	for j := 0; j < dim; j++ {
		dummyEmbedding[j] = 0.5 * float32(j+1)
	}
	return dummyEmbedding, nil
}

func (e *OpenAIEmbedder) GetDimension(_ context.Context) (int, error) {
	switch e.config.Model {
	case "text-embedding-ada-002":
		return Ada002Dimension, nil
	case "text-embedding-3-small":
		return 1536, nil
	case "text-embedding-3-large":
		return 3072, nil
	default:
		fmt.Printf("Warning: Unknown OpenAI embedding model 	%s	, defaulting dimension to %d. Consider updating GetDimension.\n", e.config.Model, Ada002Dimension)
		return Ada002Dimension, nil
	}
}

var _ iface.Embedder = (*OpenAIEmbedder)(nil)
