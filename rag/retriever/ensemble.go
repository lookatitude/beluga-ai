package retriever

import (
	"context"
	"fmt"
	"sort"

	"github.com/lookatitude/beluga-ai/schema"
)

// FusionStrategy combines multiple result sets into a single ranked list.
type FusionStrategy interface {
	// Fuse merges multiple result sets into a single ranked list of documents
	// with updated Score fields.
	Fuse(ctx context.Context, results [][]schema.Document) ([]schema.Document, error)
}

// RRFStrategy implements Reciprocal Rank Fusion. Each document's fused score
// is the sum of 1/(k + rank) across all result sets that contain it.
type RRFStrategy struct {
	// K is the RRF constant. Higher values reduce the influence of high
	// rankings. The standard default is 60.
	K int
}

// NewRRFStrategy creates an RRF fusion strategy with the given k parameter.
// Pass 0 to use the default value of 60.
func NewRRFStrategy(k int) *RRFStrategy {
	if k <= 0 {
		k = 60
	}
	return &RRFStrategy{K: k}
}

// Fuse computes RRF scores across all result sets and returns documents
// sorted by descending fused score.
func (s *RRFStrategy) Fuse(_ context.Context, results [][]schema.Document) ([]schema.Document, error) {
	scores := make(map[string]float64)
	docs := make(map[string]schema.Document)

	for _, resultSet := range results {
		for rank, doc := range resultSet {
			id := doc.ID
			scores[id] += 1.0 / float64(s.K+rank+1)
			if _, ok := docs[id]; !ok {
				docs[id] = doc
			}
		}
	}

	fused := make([]schema.Document, 0, len(docs))
	for id, doc := range docs {
		doc.Score = scores[id]
		fused = append(fused, doc)
	}

	sort.Slice(fused, func(i, j int) bool {
		return fused[i].Score > fused[j].Score
	})

	return fused, nil
}

// WeightedStrategy combines results using weighted scores. Each retriever's
// results are scaled by the corresponding weight before fusion.
type WeightedStrategy struct {
	// Weights assigns a weight to each retriever. Must have the same length
	// as the number of retrievers in the ensemble.
	Weights []float64
}

// NewWeightedStrategy creates a weighted fusion strategy with the given
// weights. Weights are normalised internally so they sum to 1.
func NewWeightedStrategy(weights []float64) *WeightedStrategy {
	return &WeightedStrategy{Weights: weights}
}

// Fuse computes weighted scores and returns documents sorted by descending
// fused score.
func (s *WeightedStrategy) Fuse(_ context.Context, results [][]schema.Document) ([]schema.Document, error) {
	if len(s.Weights) != len(results) {
		return nil, fmt.Errorf("retriever: weighted fusion: %d weights for %d result sets", len(s.Weights), len(results))
	}

	// Normalise weights.
	var total float64
	for _, w := range s.Weights {
		total += w
	}
	if total == 0 {
		return nil, fmt.Errorf("retriever: weighted fusion: weights sum to zero")
	}

	scores := make(map[string]float64)
	docs := make(map[string]schema.Document)

	for i, resultSet := range results {
		w := s.Weights[i] / total
		for _, doc := range resultSet {
			scores[doc.ID] += doc.Score * w
			if _, ok := docs[doc.ID]; !ok {
				docs[doc.ID] = doc
			}
		}
	}

	fused := make([]schema.Document, 0, len(docs))
	for id, doc := range docs {
		doc.Score = scores[id]
		fused = append(fused, doc)
	}

	sort.Slice(fused, func(i, j int) bool {
		return fused[i].Score > fused[j].Score
	})

	return fused, nil
}

// EnsembleRetriever combines multiple retrievers using a fusion strategy.
// This is the standard approach for ensemble retrieval (e.g. combining
// vector + BM25 with RRF).
type EnsembleRetriever struct {
	retrievers []Retriever
	strategy   FusionStrategy
	hooks      Hooks
}

// EnsembleOption configures an EnsembleRetriever.
type EnsembleOption func(*EnsembleRetriever)

// WithEnsembleHooks sets hooks on the EnsembleRetriever.
func WithEnsembleHooks(h Hooks) EnsembleOption {
	return func(r *EnsembleRetriever) {
		r.hooks = h
	}
}

// NewEnsembleRetriever creates a retriever that queries all inner retrievers
// and fuses results using the given strategy. If strategy is nil, RRF with
// k=60 is used.
func NewEnsembleRetriever(retrievers []Retriever, strategy FusionStrategy, opts ...EnsembleOption) *EnsembleRetriever {
	if strategy == nil {
		strategy = NewRRFStrategy(60)
	}
	r := &EnsembleRetriever{
		retrievers: retrievers,
		strategy:   strategy,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Retrieve queries all inner retrievers and fuses results.
func (r *EnsembleRetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	if r.hooks.BeforeRetrieve != nil {
		if err := r.hooks.BeforeRetrieve(ctx, query); err != nil {
			return nil, err
		}
	}

	results := make([][]schema.Document, len(r.retrievers))
	for i, ret := range r.retrievers {
		docs, err := ret.Retrieve(ctx, query, opts...)
		if err != nil {
			return nil, fmt.Errorf("retriever: ensemble retriever %d: %w", i, err)
		}
		results[i] = docs
	}

	cfg := ApplyOptions(opts...)
	fused, err := r.strategy.Fuse(ctx, results)
	if err != nil {
		return nil, fmt.Errorf("retriever: ensemble fuse: %w", err)
	}

	if cfg.TopK > 0 && len(fused) > cfg.TopK {
		fused = fused[:cfg.TopK]
	}

	if r.hooks.AfterRetrieve != nil {
		r.hooks.AfterRetrieve(ctx, fused, nil)
	}

	return fused, nil
}
