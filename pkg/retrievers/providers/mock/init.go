package mock

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/retrievers"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Config holds configuration for the MockRetriever.
type Config struct {
	// Name is the name of the mock retriever.
	Name string

	// Documents is the initial set of documents.
	Documents []schema.Document

	// DefaultK is the number of documents to return.
	DefaultK int

	// ScoreThreshold is the minimum score threshold.
	ScoreThreshold float32
}

func init() {
	// Register mock provider with the global registry
	retrievers.Register("mock", func(ctx context.Context, config any) (core.Retriever, error) {
		cfg, ok := config.(Config)
		if !ok {
			cfgPtr, okPtr := config.(*Config)
			if !okPtr {
				return nil, fmt.Errorf("invalid config type: expected Config or *Config, got %T", config)
			}
			cfg = *cfgPtr
		}

		var opts []MockOption
		if cfg.DefaultK > 0 {
			opts = append(opts, WithDefaultK(cfg.DefaultK))
		}
		if cfg.ScoreThreshold > 0 {
			opts = append(opts, WithScoreThreshold(cfg.ScoreThreshold))
		}

		return NewMockRetriever(cfg.Name, cfg.Documents, opts...), nil
	})
}
