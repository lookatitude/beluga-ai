package loader

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/schema"
)

// PipelineOption configures a LoaderPipeline.
type PipelineOption func(*LoaderPipeline)

// WithLoader adds a DocumentLoader to the pipeline.
func WithLoader(l DocumentLoader) PipelineOption {
	return func(p *LoaderPipeline) {
		p.loaders = append(p.loaders, l)
	}
}

// WithTransformer adds a Transformer to the pipeline.
func WithTransformer(t Transformer) PipelineOption {
	return func(p *LoaderPipeline) {
		p.transformers = append(p.transformers, t)
	}
}

// LoaderPipeline chains multiple loaders and transformers. When Load is called,
// all loaders are invoked and their results are concatenated. Then all
// transformers are applied to each document in order.
type LoaderPipeline struct {
	loaders      []DocumentLoader
	transformers []Transformer
}

// NewPipeline creates a new LoaderPipeline with the given options.
func NewPipeline(opts ...PipelineOption) *LoaderPipeline {
	p := &LoaderPipeline{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Load invokes all registered loaders on the source, concatenates the
// resulting documents, and applies all transformers to each document.
func (p *LoaderPipeline) Load(ctx context.Context, source string) ([]schema.Document, error) {
	if len(p.loaders) == 0 {
		return nil, fmt.Errorf("loader: pipeline has no loaders")
	}

	var docs []schema.Document
	for i, l := range p.loaders {
		result, err := l.Load(ctx, source)
		if err != nil {
			return nil, fmt.Errorf("loader: pipeline loader %d: %w", i, err)
		}
		docs = append(docs, result...)
	}

	for i, t := range p.transformers {
		transformed := make([]schema.Document, 0, len(docs))
		for _, doc := range docs {
			d, err := t.Transform(ctx, doc)
			if err != nil {
				return nil, fmt.Errorf("loader: pipeline transformer %d: %w", i, err)
			}
			transformed = append(transformed, d)
		}
		docs = transformed
	}

	return docs, nil
}
