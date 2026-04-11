package raptor

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/rag/retriever"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	// RAPTOR requires a pre-built *Tree and an Embedder, neither of which can
	// be sourced from a generic ProviderConfig. Register a factory that
	// returns a descriptive error directing callers to NewRAPTORRetriever.
	retriever.Register("raptor", func(cfg config.ProviderConfig) (retriever.Retriever, error) {
		return nil, fmt.Errorf("raptor: use NewRAPTORRetriever directly; tree and embedder cannot be provided via ProviderConfig")
	})
}

// RAPTORRetriever implements retriever.Retriever using collapsed tree search.
// It flattens all tree nodes across all levels and performs cosine similarity
// against the query embedding to find the most relevant nodes.
type RAPTORRetriever struct {
	tree     *Tree
	embedder embedding.Embedder
	topK     int
	hooks    retriever.Hooks
}

// Compile-time interface check.
var _ retriever.Retriever = (*RAPTORRetriever)(nil)

// RAPTOROption configures a RAPTORRetriever.
type RAPTOROption func(*RAPTORRetriever)

// WithTree sets the pre-built RAPTOR tree for retrieval.
func WithTree(t *Tree) RAPTOROption {
	return func(r *RAPTORRetriever) {
		r.tree = t
	}
}

// WithRetrieverEmbedder sets the embedder used to embed queries at retrieval
// time.
func WithRetrieverEmbedder(e embedding.Embedder) RAPTOROption {
	return func(r *RAPTORRetriever) {
		r.embedder = e
	}
}

// WithRaptorTopK sets the default number of results to return.
func WithRaptorTopK(k int) RAPTOROption {
	return func(r *RAPTORRetriever) {
		r.topK = k
	}
}

// WithRaptorHooks sets hooks on the RAPTORRetriever.
func WithRaptorHooks(h retriever.Hooks) RAPTOROption {
	return func(r *RAPTORRetriever) {
		r.hooks = h
	}
}

// NewRAPTORRetriever creates a new RAPTORRetriever with the given options.
func NewRAPTORRetriever(opts ...RAPTOROption) *RAPTORRetriever {
	r := &RAPTORRetriever{
		topK: 10,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Retrieve performs collapsed tree search: it embeds the query, computes
// cosine similarity against all nodes in the tree, and returns the top-K
// most similar nodes as documents.
func (r *RAPTORRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
	cfg := retriever.DefaultConfig()
	cfg.TopK = r.topK
	for _, o := range opts {
		o(&cfg)
	}

	if r.hooks.BeforeRetrieve != nil {
		if err := r.hooks.BeforeRetrieve(ctx, query); err != nil {
			return nil, err
		}
	}

	if r.tree == nil || len(r.tree.Nodes) == 0 {
		err := fmt.Errorf("raptor: retrieve: tree is empty or not set")
		if r.hooks.AfterRetrieve != nil {
			r.hooks.AfterRetrieve(ctx, nil, err)
		}
		return nil, err
	}

	if r.embedder == nil {
		err := fmt.Errorf("raptor: retrieve: embedder is required")
		if r.hooks.AfterRetrieve != nil {
			r.hooks.AfterRetrieve(ctx, nil, err)
		}
		return nil, err
	}

	queryEmb, err := r.embedder.EmbedSingle(ctx, query)
	if err != nil {
		err = fmt.Errorf("raptor: retrieve: embed query: %w", err)
		if r.hooks.AfterRetrieve != nil {
			r.hooks.AfterRetrieve(ctx, nil, err)
		}
		return nil, err
	}

	// Collapsed tree search: score all nodes across all levels.
	type scored struct {
		node  *TreeNode
		score float64
	}

	allNodes := r.tree.AllNodes()
	scored_ := make([]scored, 0, len(allNodes))
	for _, node := range allNodes {
		sim := cosineSimilarity(queryEmb, node.Embedding)
		if cfg.Threshold > 0 && sim < cfg.Threshold {
			continue
		}
		scored_ = append(scored_, scored{node: node, score: sim})
	}

	// Sort by descending score.
	sort.Slice(scored_, func(i, j int) bool {
		return scored_[i].score > scored_[j].score
	})

	topK := cfg.TopK
	if topK > len(scored_) {
		topK = len(scored_)
	}

	docs := make([]schema.Document, topK)
	for i := 0; i < topK; i++ {
		s := scored_[i]
		metadata := make(map[string]any)
		for k, v := range s.node.Metadata {
			metadata[k] = v
		}
		metadata["raptor_level"] = s.node.Level
		metadata["raptor_node_id"] = s.node.ID

		docs[i] = schema.Document{
			ID:       s.node.ID,
			Content:  s.node.Content,
			Score:    s.score,
			Metadata: metadata,
		}
	}

	if r.hooks.AfterRetrieve != nil {
		r.hooks.AfterRetrieve(ctx, docs, nil)
	}

	return docs, nil
}

// cosineSimilarity computes the cosine similarity between two vectors.
// Returns 0 if either vector has zero magnitude.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		ai, bi := float64(a[i]), float64(b[i])
		dot += ai * bi
		normA += ai * ai
		normB += bi * bi
	}

	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}

	return dot / denom
}
