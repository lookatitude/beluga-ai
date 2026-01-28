// Package retrievers provides built-in provider registrations.
// This file registers the standard retriever providers with the global registry.
package retrievers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

func init() {
	// Register vectorstore provider
	Register("vectorstore", func(ctx context.Context, config any) (core.Retriever, error) {
		cfg, ok := config.(*VectorStoreProviderConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type: expected *VectorStoreProviderConfig, got %T", config)
		}
		return NewVectorStoreRetriever(cfg.VectorStore,
			WithDefaultK(cfg.K),
			WithScoreThreshold(cfg.ScoreThreshold),
			WithTimeout(cfg.Timeout),
			WithTracing(cfg.EnableTracing),
			WithMetrics(cfg.EnableMetrics),
		)
	})

	// Register multiquery provider
	Register("multiquery", func(ctx context.Context, config any) (core.Retriever, error) {
		cfg, ok := config.(*MultiQueryProviderConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type: expected *MultiQueryProviderConfig, got %T", config)
		}
		return NewMultiQueryRetriever(cfg.Retriever, cfg.LLM,
			WithDefaultK(cfg.NumQueries),
			WithTracing(cfg.EnableTracing),
			WithMetrics(cfg.EnableMetrics),
		)
	})

	// Register mock provider
	Register("mock", func(ctx context.Context, config any) (core.Retriever, error) {
		cfg, ok := config.(*MockProviderConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type: expected *MockProviderConfig, got %T", config)
		}
		// Use the mock retriever from providers/mock package
		// For now, return a simple implementation
		return &mockRetrieverWrapper{
			name:      cfg.Name,
			documents: cfg.Documents,
			defaultK:  cfg.DefaultK,
		}, nil
	})
}

// VectorStoreProviderConfig is the configuration for creating a VectorStoreRetriever via the registry.
type VectorStoreProviderConfig struct {
	VectorStore    vectorstores.VectorStore
	K              int
	ScoreThreshold float32
	Timeout        time.Duration
	EnableTracing  bool
	EnableMetrics  bool
}

// MultiQueryProviderConfig is the configuration for creating a MultiQueryRetriever via the registry.
type MultiQueryProviderConfig struct {
	Retriever     core.Retriever
	LLM           llmsiface.ChatModel
	NumQueries    int
	EnableTracing bool
	EnableMetrics bool
}

// MockProviderConfig is the configuration for creating a mock retriever via the registry.
type MockProviderConfig struct {
	Name           string
	Documents      []schema.Document
	DefaultK       int
	ScoreThreshold float32
}

// mockRetrieverWrapper is a simple mock retriever for testing.
type mockRetrieverWrapper struct {
	name      string
	documents []schema.Document
	defaultK  int
}

func (m *mockRetrieverWrapper) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	k := m.defaultK
	if k <= 0 {
		k = 4
	}
	if k > len(m.documents) {
		k = len(m.documents)
	}
	return m.documents[:k], nil
}

func (m *mockRetrieverWrapper) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	query, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("expected string input, got %T", input)
	}
	return m.GetRelevantDocuments(ctx, query)
}

func (m *mockRetrieverWrapper) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := m.Invoke(ctx, input, options...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (m *mockRetrieverWrapper) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	return nil, errors.New("streaming not supported")
}

var (
	_ core.Retriever = (*mockRetrieverWrapper)(nil)
	_ core.Runnable  = (*mockRetrieverWrapper)(nil)
)
